package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strconv"
	"time"
)

// AddComment 添加评论
func AddComment(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id       string `json:"_id"`
		Content  string `json:"content"`
		QQ       string `json:"qq"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	comment := models.Comment{
		BlogID:    id,
		Username:  data.Username,
		QQ:        data.QQ,
		Content:   data.Content,
		CreatedAt: time.Now(),
		Like:      0,
	}

	_, err := db.InsertOne(context.Background(), comment)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"err": "评论插入失败" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论插入成功"})

}

// ResetComments 编辑评论
func ResetComments(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id      string `json:"_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	_, err := db.UpdateByID(context.Background(), id, bson.M{
		"$set": bson.M{"content": data.Content},
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"err": "评论编辑失败" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论编辑成功"})
}

// DeleteComments 删除评论
func DeleteComments(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	_, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"err": "评论删除失败" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论删除成功"})
}

func LikeComment(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	err1 := db.FindOne(context.Background(), bson.M{"_id": id})

	if err1.Err() != nil {
		c.JSON(http.StatusOK, gin.H{"error": "该评论不存在"})
		return
	}

	_, err := db.UpdateByID(context.Background(), id, bson.M{
		"$inc": bson.M{
			"like": 1,
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "点赞失败 " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论点赞成功"})
}

func UnLikeComment(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	var comment models.Comment

	err1 := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&comment)

	if err1 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "该评论不存在"})
		return
	}

	if comment.Like <= 0 {
		c.JSON(http.StatusOK, gin.H{"error": "点赞数量不能为负数"})
		return
	}

	_, err := db.UpdateByID(context.Background(), id, bson.M{
		"$inc": bson.M{
			"like": -1,
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "点赞失败 " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论取消点赞成功"})
}

func GetComment(c *gin.Context) {
	db := utils.GetCollection("comments")

	blogStr := c.Param("blog")
	pageStr := c.Param("page")
	limitStr := c.Param("limit")

	blog, _ := primitive.ObjectIDFromHex(blogStr)

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "无效的页码"})
		return
	}

	limit, err1 := strconv.Atoi(limitStr)
	if err1 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "无效的每页数量"})
		return
	}

	if page <= 0 || limit <= 0 {
		c.JSON(http.StatusOK, gin.H{"error": "无效的查询"})
		return
	}

	skip := (page - 1) * limit

	var comments []models.Comment

	option := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err2 := db.Find(context.Background(), bson.M{"blogId": blog}, option)
	if err2 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "查询失败: " + err2.Error()})
		return
	}
	defer cursor.Close(context.Background())

	// 遍历游标并解码每个文档
	for cursor.Next(context.Background()) {
		var comment models.Comment
		if err3 := cursor.Decode(&comment); err3 != nil {
			c.JSON(http.StatusOK, gin.H{"error": "解码失败: " + err3.Error()})
			return
		}
		comments = append(comments, comment)
	}

	// 检查游标遍历过程中的错误
	if err4 := cursor.Err(); err4 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "游标错误: " + err4.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})

}

func GetCommentCount(c *gin.Context) {

	db := utils.GetCollection("comments")

	blogStr := c.Param("blog")

	blog, _ := primitive.ObjectIDFromHex(blogStr)

	count, err := db.CountDocuments(context.Background(), bson.M{"blogId": blog})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "搜索评论总数出现错误, 错误原因: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}
