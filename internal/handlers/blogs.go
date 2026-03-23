package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// updateTagCount 是一个辅助函数，用于更新单个标签的引用计数
// increment 可以是正数（增加）或负数（减少）
func updateTagCount(tagsDB *mongo.Collection, tagName string, increment int) error {
	// 定义更新操作
	update := bson.M{"$inc": bson.M{"count": increment}}
	// 定义查询条件
	filter := bson.M{"name": tagName}

	// 尝试更新标签。Upsert: true 表示如果找不到标签，就创建一个新的
	// 这样做可以简化代码，将“创建新标签”和“更新旧标签”合并为一个操作
	opts := options.Update().SetUpsert(true)
	_, err := tagsDB.UpdateOne(context.Background(), filter, update, opts)

	// 如果 increment 是正数，即使 upsert 也能正常工作。
	// 如果是负数，而标签不存在，它会创建一个 count 为负数的标签，这在逻辑上是错误的。
	// 所以在调用此函数之前，应该确保在减少计数时标签是存在的。
	// 在 ReSetBlog 和 DeleteBlog 中，我们已经先查询了博客，所以我们知道旧标签是存在的。

	return err
}

// InsertBlog 插入一篇新博客
func InsertBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var blog models.Blog
	// 绑定并验证请求的 JSON 数据
	if err := c.ShouldBindJSON(&blog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 设置创建和更新时间
	blog.UpdatedAt = time.Now()
	blog.CreatedAt = time.Now()
	blog.Views = 0 // 新文章浏览量为0

	// 更新关联的标签计数
	tagsDB := utils.GetCollection("tags")
	for _, tag := range blog.Tags {
		if err := updateTagCount(tagsDB, tag, 1); err != nil {
			// 如果更新标签失败，返回错误，不继续插入博客
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新标签计数失败: " + err.Error()})
			return
		}
	}

	// 将博客插入数据库
	// 注意：这里没有使用事务，如果博客插入失败，之前对标签的计数更新不会回滚。
	// 在高并发或需要强一致性的场景下，应使用 MongoDB 事务。
	result, err := db.InsertOne(context.Background(), blog)
	if err != nil {
		// BUG修复：之前这里错误地使用了 `err.Error()`，它可能来自之前的操作，导致潜在的 nil 指针解引用。
		// 已修正为使用当前操作返回的 `err`。
		c.JSON(http.StatusInternalServerError, gin.H{"error": "插入博客失败: " + err.Error()})
		// 理论上这里应该回滚标签计数，但为了简化，我们暂时只记录错误。
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "插入成功",
		"_id":     result.InsertedID,
	})
}

// ViewAdd 阅读量增加 (此函数似乎未被路由使用，但我们仍然修复其中的bug)
func ViewAdd(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// BUG修复：之前忽略了 ObjectIDFromHex 可能返回的错误。
	// 如果 ID 格式不正确，会导致后续操作失败。
	id, err := primitive.ObjectIDFromHex(data.Id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	update := bson.M{
		"$inc": bson.M{"views": 1},
	}

	// 使用 UpdateByID 更新浏览量
	_, err = db.UpdateByID(context.Background(), id, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败, 原因:" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ReSetBlog 编辑博客
func ReSetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var updatedBlog models.Blog
	if err := c.ShouldBindJSON(&updatedBlog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查 ID 格式是否有效
	id, err := primitive.ObjectIDFromHex(updatedBlog.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	// 1. 查找旧博客以获取旧的标签列表
	var oldBlog models.Blog
	err = db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&oldBlog)
	if err != nil {
		// 如果找不到博客，返回错误
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到要更新的博客"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询原博客失败: " + err.Error()})
		return
	}

	// 2. 计算需要更新的标签
	// 使用 map 来高效地计算新旧标签的差异
	oldTagsMap := make(map[string]bool)
	for _, tag := range oldBlog.Tags {
		oldTagsMap[tag] = true
	}

	newTagsMap := make(map[string]bool)
	for _, tag := range updatedBlog.Tags {
		newTagsMap[tag] = true
	}

	tagsDB := utils.GetCollection("tags")

	// 减少被移除的旧标签的计数
	for tag := range oldTagsMap {
		if !newTagsMap[tag] {
			if err := updateTagCount(tagsDB, tag, -1); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("减少旧标签 '%s' 计数失败: %s", tag, err.Error())})
				return
			}
		}
	}

	// 增加新增的标签的计数
	for tag := range newTagsMap {
		if !oldTagsMap[tag] {
			if err := updateTagCount(tagsDB, tag, 1); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("增加新标签 '%s' 计数失败: %s", tag, err.Error())})
				return
			}
		}
	}

	// 3. 更新博客内容
	update := bson.M{
		"$set": bson.M{
			"title":     updatedBlog.Title,
			"tags":      updatedBlog.Tags,
			"content":   updatedBlog.Content,
			"updatedAt": time.Now(),
			"views":     oldBlog.Views,
		},
	}

	_, err = db.UpdateByID(context.Background(), id, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新博客失败: " + err.Error()})
		// 注意：此处如果更新失败，标签计数已经修改，存在数据不一致的风险。生产环境应使用事务。
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// GetBlog 获取单篇博客详情，并在成功访问后将阅读量加 1
func GetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	blogIdStr := c.Param("blogId")

	id, err := primitive.ObjectIDFromHex(blogIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	update := bson.M{"$inc": bson.M{"views": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var blog blogDetail
	err = db.FindOneAndUpdate(context.Background(), bson.M{"_id": id}, update, opts).Decode(&blog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此博客"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询博客失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"_id":       blog.ID.Hex(),
			"title":     blog.Title,
			"content":   blog.Content,
			"tags":      blog.Tags,
			"createdAt": blog.CreatedAt,
			"updatedAt": blog.UpdatedAt,
			"views":     blog.Views,
		},
	})
}

// DeleteBlog 删除博客
func DeleteBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查 ID 格式是否有效
	id, err := primitive.ObjectIDFromHex(data.Id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	// 1. 先查找要删除的博客，以获取其标签列表
	var blog models.Blog
	err = db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&blog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到要删除的博客"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询原博客失败: " + err.Error()})
		return
	}

	// 2. 删除博客文档
	_, err = db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除博客失败: " + err.Error()})
		return
	}

	// 3. 博客成功删除后，再更新标签计数
	// 这种顺序的好处是，如果博客删除失败，我们根本不会去动标签。
	// 如果博客删除成功但标签更新失败，虽然会留下未更新的标签计数，但这比博客还在但标签计数已减少要好。
	tagsDB := utils.GetCollection("tags")
	for _, tag := range blog.Tags {
		if err := updateTagCount(tagsDB, tag, -1); err != nil {
			// 即使某个标签更新失败，也应该继续尝试更新其他标签
			// 在生产环境中，这里应该记录详细日志
			fmt.Printf("警告: 删除博客 %s 后，减少标签 '%s' 计数失败: %v\n", id.Hex(), tag, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功删除博客: " + blog.Title})
}

// rep 是用于 GetBlogs 接口返回的博客列表的简化结构体
type rep struct {
	ID           string    `json:"_id"` // BSON的omitempty在这里不需要
	Title        string    `json:"title"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Views        int       `json:"views"`
	CommentCount int64     `json:"commentCount"`
}

type blogDetail struct {
	ID        primitive.ObjectID `bson:"_id"`
	Title     string             `bson:"title"`
	Content   string             `bson:"content"`
	Tags      []string           `bson:"tags"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
	Views     int                `bson:"views"`
}

type pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func getCommentCountsByBlogIDs(blogIDs []primitive.ObjectID) (map[string]int64, error) {
	commentCounts := make(map[string]int64, len(blogIDs))
	if len(blogIDs) == 0 {
		return commentCounts, nil
	}

	commentsDB := utils.GetCollection("comments")
	cursor, err := commentsDB.Aggregate(context.Background(), mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"blogId": bson.M{"$in": blogIDs}}}},
		{{Key: "$group", Value: bson.M{"_id": "$blogId", "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(context.Background())
	}()

	var result []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Count int64              `bson:"count"`
	}
	if err = cursor.All(context.Background(), &result); err != nil {
		return nil, err
	}

	for _, item := range result {
		commentCounts[item.ID.Hex()] = item.Count
	}

	return commentCounts, nil
}

// GetBlogs 获取博客列表（分页）
func GetBlogs(c *gin.Context) {
	db := utils.GetCollection("blogs")

	// 从 URL 参数获取分页信息
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1 // 提供一个安全的默认值
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10 // 提供一个安全的默认值
	}

	skip := (page - 1) * limit

	// 设置查询选项：排序、跳过、限制
	// 按创建时间降序排序，使最新的文章显示在前面
	opts := options.Find().SetSort(bson.D{{"createdAt", -1}}).SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}
	// 改进：在 defer 中检查 Close() 的错误
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			// 生产环境应使用日志库
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var blogs []rep
	var blogIDs []primitive.ObjectID
	// 遍历游标
	for cursor.Next(context.Background()) {
		var blog models.Blog
		if err := cursor.Decode(&blog); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解码博客数据失败: " + err.Error()})
			return
		}

		blogObjectID, err := primitive.ObjectIDFromHex(blog.ID)
		if err == nil {
			blogIDs = append(blogIDs, blogObjectID)
		}

		blogs = append(blogs, rep{
			ID:        blog.ID, // 在 model 中 ID 已经是 string 了
			Title:     blog.Title,
			Tags:      blog.Tags,
			CreatedAt: blog.CreatedAt,
			UpdatedAt: blog.UpdatedAt,
			Views:     blog.Views,
		})
	}

	// 检查游标在迭代过程中是否出错
	if err := cursor.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "游标错误: " + err.Error()})
		return
	}

	// 同时返回总数，便于前端分页
	total, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询博客总数失败: " + err.Error()})
		return
	}

	commentCounts, err := getCommentCountsByBlogIDs(blogIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询评论数量失败: " + err.Error()})
		return
	}

	for i := range blogs {
		if count, ok := commentCounts[blogs[i].ID]; ok {
			blogs[i].CommentCount = count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"list": blogs,
		"pagination": pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetBlogCount 获取博客总数
func GetBlogCount(c *gin.Context) {
	db := utils.GetCollection("blogs")
	count, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询博客数量失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// SearchBlogs 搜索博客
func SearchBlogs(c *gin.Context) {
	db := utils.GetCollection("blogs")

	// 改进：从 query 参数获取搜索词，这更符合 RESTful 风格
	searchStr := c.Query("q")
	if searchStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入搜索关键词"})
		return
	}

	// 使用正则表达式进行模糊匹配，'i' 选项表示不区分大小写
	filter := bson.D{
		{"$or", bson.A{
			bson.D{{"title", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
			bson.D{{"tags", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
			bson.D{{"content", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
		}},
	}

	// 按相关性或时间排序
	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})

	cursor, err := db.Find(context.Background(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var blogs []rep
	if err = cursor.All(context.Background(), &blogs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解码博客数据失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keyword": searchStr,
		"total":   len(blogs),
		"list":    blogs,
	})
}
