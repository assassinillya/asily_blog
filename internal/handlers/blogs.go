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
	update := bson.M{"$inc": bson.M{"count": increment}}
	filter := bson.M{"name": tagName}

	opts := options.Update().SetUpsert(true)
	_, err := tagsDB.UpdateOne(context.Background(), filter, update, opts)

	return err
}

// InsertBlog 插入一篇新博客
func InsertBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var blog models.Blog
	if err := c.ShouldBindJSON(&blog); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	blog.UpdatedAt = time.Now()
	blog.CreatedAt = time.Now()
	blog.Views = 0

	tagsDB := utils.GetCollection("tags")
	for _, tag := range blog.Tags {
		if err := updateTagCount(tagsDB, tag, 1); err != nil {
			respondError(c, http.StatusInternalServerError, "更新标签计数失败: "+err.Error())
			return
		}
	}

	result, err := db.InsertOne(context.Background(), blog)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "插入博客失败: "+err.Error())
		return
	}

	respondMessageData(c, http.StatusOK, "插入成功", gin.H{
		"_id": result.InsertedID,
	})
}

// ViewAdd 阅读量增加
func ViewAdd(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.Id)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	update := bson.M{
		"$inc": bson.M{"views": 1},
	}

	_, err = db.UpdateByID(context.Background(), id, update)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "更新失败, 原因:"+err.Error())
		return
	}

	respondMessage(c, http.StatusOK, "更新成功")
}

// ReSetBlog 编辑博客
func ReSetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var updatedBlog models.Blog
	if err := c.ShouldBindJSON(&updatedBlog); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(updatedBlog.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	var oldBlog models.Blog
	err = db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&oldBlog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondError(c, http.StatusNotFound, "找不到要更新的博客")
			return
		}
		respondError(c, http.StatusInternalServerError, "查询原博客失败: "+err.Error())
		return
	}

	oldTagsMap := make(map[string]bool)
	for _, tag := range oldBlog.Tags {
		oldTagsMap[tag] = true
	}

	newTagsMap := make(map[string]bool)
	for _, tag := range updatedBlog.Tags {
		newTagsMap[tag] = true
	}

	tagsDB := utils.GetCollection("tags")
	for tag := range oldTagsMap {
		if !newTagsMap[tag] {
			if err := updateTagCount(tagsDB, tag, -1); err != nil {
				respondError(c, http.StatusInternalServerError, fmt.Sprintf("减少旧标签 '%s' 计数失败: %s", tag, err.Error()))
				return
			}
		}
	}

	for tag := range newTagsMap {
		if !oldTagsMap[tag] {
			if err := updateTagCount(tagsDB, tag, 1); err != nil {
				respondError(c, http.StatusInternalServerError, fmt.Sprintf("增加新标签 '%s' 计数失败: %s", tag, err.Error()))
				return
			}
		}
	}

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
		respondError(c, http.StatusInternalServerError, "更新博客失败: "+err.Error())
		return
	}

	respondMessage(c, http.StatusOK, "更新成功")
}

// GetBlog 获取单篇博客详情，并在成功访问后将阅读量加 1
func GetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	blogIdStr := c.Param("blogId")

	id, err := primitive.ObjectIDFromHex(blogIdStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	update := bson.M{"$inc": bson.M{"views": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var blog blogDetail
	err = db.FindOneAndUpdate(context.Background(), bson.M{"_id": id}, update, opts).Decode(&blog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondError(c, http.StatusNotFound, "找不到此博客")
			return
		}
		respondError(c, http.StatusInternalServerError, "查询博客失败: "+err.Error())
		return
	}

	respondData(c, http.StatusOK, gin.H{
		"_id":       blog.ID.Hex(),
		"title":     blog.Title,
		"content":   blog.Content,
		"tags":      blog.Tags,
		"createdAt": blog.CreatedAt,
		"updatedAt": blog.UpdatedAt,
		"views":     blog.Views,
	})
}

// DeleteBlog 删除博客
func DeleteBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.Id)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	var blog models.Blog
	err = db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&blog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondError(c, http.StatusNotFound, "找不到要删除的博客")
			return
		}
		respondError(c, http.StatusInternalServerError, "查询原博客失败: "+err.Error())
		return
	}

	_, err = db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "删除博客失败: "+err.Error())
		return
	}

	tagsDB := utils.GetCollection("tags")
	for _, tag := range blog.Tags {
		if err := updateTagCount(tagsDB, tag, -1); err != nil {
			fmt.Printf("警告: 删除博客 %s 后，减少标签 '%s' 计数失败: %v\n", id.Hex(), tag, err)
		}
	}

	respondMessage(c, http.StatusOK, "成功删除博客: "+blog.Title)
}

type rep struct {
	ID           string    `json:"_id"`
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

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	skip := (page - 1) * limit
	opts := options.Find().SetSort(bson.D{{"createdAt", -1}}).SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var blogs []rep
	var blogIDs []primitive.ObjectID
	for cursor.Next(context.Background()) {
		var blog models.Blog
		if err := cursor.Decode(&blog); err != nil {
			respondError(c, http.StatusInternalServerError, "解码博客数据失败: "+err.Error())
			return
		}

		blogObjectID, err := primitive.ObjectIDFromHex(blog.ID)
		if err == nil {
			blogIDs = append(blogIDs, blogObjectID)
		}

		blogs = append(blogs, rep{
			ID:        blog.ID,
			Title:     blog.Title,
			Tags:      blog.Tags,
			CreatedAt: blog.CreatedAt,
			UpdatedAt: blog.UpdatedAt,
			Views:     blog.Views,
		})
	}

	if err := cursor.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "游标错误: "+err.Error())
		return
	}

	total, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询博客总数失败: "+err.Error())
		return
	}

	commentCounts, err := getCommentCountsByBlogIDs(blogIDs)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询评论数量失败: "+err.Error())
		return
	}

	for i := range blogs {
		if count, ok := commentCounts[blogs[i].ID]; ok {
			blogs[i].CommentCount = count
		}
	}

	respondList(c, blogs, &pagination{
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil)
}

// GetBlogCount 获取博客总数
func GetBlogCount(c *gin.Context) {
	db := utils.GetCollection("blogs")
	count, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询博客数量失败: "+err.Error())
		return
	}

	respondData(c, http.StatusOK, gin.H{"count": count})
}

// SearchBlogs 搜索博客
func SearchBlogs(c *gin.Context) {
	db := utils.GetCollection("blogs")

	searchStr := c.Query("q")
	if searchStr == "" {
		respondError(c, http.StatusBadRequest, "请输入搜索关键词")
		return
	}

	filter := bson.D{
		{"$or", bson.A{
			bson.D{{"title", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
			bson.D{{"tags", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
			bson.D{{"content", bson.M{"$regex": primitive.Regex{Pattern: searchStr, Options: "i"}}}},
		}},
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})

	cursor, err := db.Find(context.Background(), filter, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var blogs []rep
	if err = cursor.All(context.Background(), &blogs); err != nil {
		respondError(c, http.StatusInternalServerError, "解码博客数据失败: "+err.Error())
		return
	}

	respondList(c, blogs, nil, gin.H{
		"keyword": searchStr,
		"total":   len(blogs),
	})
}
