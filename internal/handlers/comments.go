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
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddComment 添加评论
func AddComment(c *gin.Context) {
	var data struct {
		BlogID   string `json:"blogId"` // 修正：字段名应与前端对应，通常是 blogId
		Content  string `json:"content"`
		QQ       string `json:"qq"`
		Username string `json:"username"`
	}
	// 绑定并验证请求数据
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 校验必填字段
	if data.BlogID == "" || data.Content == "" || data.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "博客ID、内容和用户名不能为空"})
		return
	}

	// BUG修复：检查博客ID格式是否有效
	blogID, err := primitive.ObjectIDFromHex(data.BlogID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	// 健壮性改进：在添加评论前，检查对应的博客是否存在
	blogsDB := utils.GetCollection("blogs")
	count, err := blogsDB.CountDocuments(context.Background(), bson.M{"_id": blogID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查博客是否存在时出错: " + err.Error()})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "评论关联的博客不存在"})
		return
	}

	// 创建评论模型
	comment := models.Comment{
		BlogID:      blogID,
		Username:    data.Username,
		QQ:          data.QQ, // QQ可以为空
		Content:     data.Content,
		CreatedAt:   time.Now(),
		Like:        0,
		LikeCount:   0,
		UnLikeCount: 0,
	}

	// 插入评论
	commentsDB := utils.GetCollection("comments")
	result, err := commentsDB.InsertOne(context.Background(), comment)
	if err != nil {
		// 状态码修正：服务器内部错误应返回 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论插入失败: " + err.Error()})
		return
	}

	insertedID, _ := result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusOK, gin.H{
		"message": "评论插入成功",
		"id":      insertedID.Hex(),
	})
}

// ResetComments 编辑评论 (通常只有管理员或评论所有者可以编辑)
func ResetComments(c *gin.Context) {
	var data struct {
		ID      string `json:"_id"` // 修正：字段名应为 ID
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查评论ID格式
	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID格式"})
		return
	}

	// 更新评论
	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$set": bson.M{"content": data.Content},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论编辑失败: " + err.Error()})
		return
	}

	// 检查是否真的找到了并更新了文档
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到要编辑的评论"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论编辑成功"})
}

// DeleteComments 删除评论 (通常只有管理员或博客所有者可以删除)
func DeleteComments(c *gin.Context) {
	var data struct {
		ID string `json:"_id"` // 修正：字段名应为 ID
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查评论ID格式
	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID格式"})
		return
	}

	// 删除评论
	db := utils.GetCollection("comments")
	result, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论删除失败: " + err.Error()})
		return
	}

	// 检查是否真的找到了并删除了文档
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到要删除的评论"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论删除成功"})
}

// LikeComment 给评论点赞
func LikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"` // 修正：字段名应为 ID
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查评论ID格式
	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID格式"})
		return
	}

	// 优化：直接使用 UpdateByID 并检查其结果，无需预先查询
	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$inc": bson.M{
			"like":      1,
			"likeCount": 1,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "点赞失败: " + err.Error()})
		return
	}

	// 如果 MatchedCount 是 0，说明没有找到该评论
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该评论不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论点赞成功"})
}

// UnLikeComment 取消评论点赞
func UnLikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"` // 修正：字段名应为 ID
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查评论ID格式
	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID格式"})
		return
	}

	db := utils.GetCollection("comments")

	// 为了防止点赞数变为负数，我们需要一个原子操作。
	// 我们只在 'like' 大于 0 的情况下才执行减少操作。
	filter := bson.M{
		"_id":  id,
		"like": bson.M{"$gt": 0}, // 只有当 like 大于 0 时才匹配
	}
	update := bson.M{
		"$inc": bson.M{
			"like":        -1,
			"unlikeCount": 1,
		},
	}

	result, err := db.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消点赞失败: " + err.Error()})
		return
	}

	// 如果 MatchedCount 是 0，有两种可能：评论不存在，或者评论的点赞数已经是0。
	if result.MatchedCount == 0 {
		// 我们可以进一步查询来确定具体原因
		var comment models.Comment
		err := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&comment)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "该评论不存在"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "点赞数已经是0，无法取消点赞"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论取消点赞成功"})
}

// GetComment 获取某篇博客下的评论列表（分页）
func GetComment(c *gin.Context) {
	db := utils.GetCollection("comments")

	// 从 URL 参数获取博客ID和分页信息
	blogStr := c.Param("blogId") // 修正：与路由定义保持一致
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	// BUG修复：检查博客ID格式
	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	// 校验分页参数
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	// 设置查询选项：按创建时间升序排序（旧评论在前）
	opts := options.Find().SetSort(bson.D{{"createdAt", 1}}).SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.M{"blogId": blogID}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}
	// 改进：在 defer 中检查 Close() 的错误
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var comments []models.Comment
	// 使用 cursor.All() 一次性解码所有文档，更简洁
	if err = cursor.All(context.Background(), &comments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解码评论数据失败: " + err.Error()})
		return
	}

	// 检查游标在迭代过程中是否出错
	if err := cursor.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "游标错误: " + err.Error()})
		return
	}

	// 如果没有评论，返回一个空数组而不是错误
	if comments == nil {
		comments = []models.Comment{}
	}

	total, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询评论总数失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"list": comments,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetCommentCount 获取某篇博客的评论总数
func GetCommentCount(c *gin.Context) {
	db := utils.GetCollection("comments")
	blogStr := c.Param("blogId") // 修正：与路由定义保持一致

	// BUG修复：检查博客ID格式
	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博客ID格式"})
		return
	}

	// 查询评论总数
	count, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索评论总数出现错误: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}
