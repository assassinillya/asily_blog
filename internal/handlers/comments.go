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
		BlogID   string `json:"blogId"`
		Content  string `json:"content"`
		QQ       string `json:"qq"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	if data.BlogID == "" || data.Content == "" || data.Username == "" {
		respondError(c, http.StatusBadRequest, "博客ID、内容和用户名不能为空")
		return
	}

	blogID, err := primitive.ObjectIDFromHex(data.BlogID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	blogsDB := utils.GetCollection("blogs")
	count, err := blogsDB.CountDocuments(context.Background(), bson.M{"_id": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "检查博客是否存在时出错: "+err.Error())
		return
	}
	if count == 0 {
		respondError(c, http.StatusNotFound, "评论关联的博客不存在")
		return
	}

	comment := models.Comment{
		BlogID:      blogID,
		Username:    data.Username,
		QQ:          data.QQ,
		Content:     data.Content,
		CreatedAt:   time.Now(),
		Like:        0,
		LikeCount:   0,
		UnLikeCount: 0,
	}

	commentsDB := utils.GetCollection("comments")
	result, err := commentsDB.InsertOne(context.Background(), comment)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论插入失败: "+err.Error())
		return
	}

	insertedID, _ := result.InsertedID.(primitive.ObjectID)
	respondMessageData(c, http.StatusOK, "评论插入成功", gin.H{"id": insertedID.Hex()})
}

// ResetComments 编辑评论
func ResetComments(c *gin.Context) {
	var data struct {
		ID      string `json:"_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$set": bson.M{"content": data.Content},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论编辑失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要编辑的评论")
		return
	}

	respondMessage(c, http.StatusOK, "评论编辑成功")
}

// DeleteComments 删除评论
func DeleteComments(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	result, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论删除失败: "+err.Error())
		return
	}

	if result.DeletedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要删除的评论")
		return
	}

	respondMessage(c, http.StatusOK, "评论删除成功")
}

// LikeComment 给评论点赞
func LikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$inc": bson.M{
			"like":      1,
			"likeCount": 1,
		},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "点赞失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "该评论不存在")
		return
	}

	respondMessage(c, http.StatusOK, "评论点赞成功")
}

// UnLikeComment 取消评论点赞
func UnLikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	filter := bson.M{
		"_id":  id,
		"like": bson.M{"$gt": 0},
	}
	update := bson.M{
		"$inc": bson.M{
			"like":        -1,
			"unlikeCount": 1,
		},
	}

	result, err := db.UpdateOne(context.Background(), filter, update)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "取消点赞失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		var comment models.Comment
		err := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&comment)
		if err != nil {
			respondError(c, http.StatusNotFound, "该评论不存在")
		} else {
			respondError(c, http.StatusBadRequest, "点赞数已经是0，无法取消点赞")
		}
		return
	}

	respondMessage(c, http.StatusOK, "评论取消点赞成功")
}

// GetComment 获取某篇博客下的评论列表（分页）
func GetComment(c *gin.Context) {
	db := utils.GetCollection("comments")

	blogStr := c.Param("blogId")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	opts := options.Find().SetSort(bson.D{{"createdAt", 1}}).SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.M{"blogId": blogID}, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var comments []models.Comment
	if err = cursor.All(context.Background(), &comments); err != nil {
		respondError(c, http.StatusInternalServerError, "解码评论数据失败: "+err.Error())
		return
	}

	if err := cursor.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "游标错误: "+err.Error())
		return
	}

	if comments == nil {
		comments = []models.Comment{}
	}

	total, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询评论总数失败: "+err.Error())
		return
	}

	respondList(c, comments, &pagination{
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil)
}

// GetCommentCount 获取某篇博客的评论总数
func GetCommentCount(c *gin.Context) {
	db := utils.GetCollection("comments")
	blogStr := c.Param("blogId")

	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	count, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "搜索评论总数出现错误: "+err.Error())
		return
	}

	respondData(c, http.StatusOK, gin.H{"count": count})
}
