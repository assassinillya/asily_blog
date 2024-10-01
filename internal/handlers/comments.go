package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

// AddComment 添加评论
func AddComment(c *gin.Context) {
	db := utils.GetCollection("comments")
	var data struct {
		Id       string `json:"_id"`
		Context  string `json:"context"`
		QQ       string `json:"qq"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)

	comment := models.Comment{
		BlogID:    id,
		Username:  data.Username,
		QQ:        data.QQ,
		Content:   data.Context,
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

// GetComments 获取所有评论
func GetComments(c *gin.Context) {

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
