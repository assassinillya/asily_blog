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
)

func AddLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")
	var data models.FriendLink

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.InsertOne(context.Background(), data)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "友情链接插入失败, 原因: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "友情链接添加成功"})
}

func DeleteLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data struct {
		Id string `json:"_id"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	// 将字符串的 _id 转换为 ObjectID
	id, err := primitive.ObjectIDFromHex(data.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "无效的ID格式"})
		return
	}

	ok := db.FindOne(context.Background(), bson.M{"_id": id})
	if ok.Err() != nil {
		c.JSON(http.StatusOK, gin.H{"error": "找不到原友情链接, 原因: " + ok.Err().Error()})
		return
	}

	// 执行删除操作
	_, err = db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "友情链接删除失败: " + err.Error()})
		return
	}

	// 返回成功消息
	c.JSON(http.StatusOK, gin.H{"message": "友情链接删除成功"})
}

func UpdateLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data models.FriendLink
	// 注意! 这里必须要带上_id

	// 绑定请求中的 JSON 数据
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	id, _ := primitive.ObjectIDFromHex(data.ID)

	ok := db.FindOne(context.Background(), bson.M{"_id": id})
	if ok.Err() != nil {
		c.JSON(http.StatusOK, gin.H{"error": "查询原友情链接失败, 失败原因: " + ok.Err().Error()})
		return
	}

	// 执行更新操作
	_, err := db.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"name":        data.Name,
			"avatar":      data.Avatar,
			"url":         data.URL,
			"description": data.Description,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "友情链接更新失败: " + err.Error()})
		return
	}

	// 返回成功消息
	c.JSON(http.StatusOK, gin.H{"message": "友情链接更新成功"})
}

func GetLinks(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	pageStr := c.Param("page")
	limitStr := c.Param("limit")

	page, err1 := strconv.Atoi(pageStr)
	if err1 != nil {
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

	var links []models.FriendLink

	option := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err1 := db.Find(context.Background(), bson.D{}, option)
	if err1 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "查询失败: " + err1.Error()})
		return
	}
	defer cursor.Close(context.Background())

	// 遍历游标并解码每个文档
	for cursor.Next(context.Background()) {
		var link models.FriendLink
		if err2 := cursor.Decode(&link); err2 != nil {
			c.JSON(http.StatusOK, gin.H{"error": "解码失败: " + err2.Error()})
			return
		}
		links = append(links, link)
	}

	// 检查游标遍历过程中的错误
	if err2 := cursor.Err(); err2 != nil {
		c.JSON(http.StatusOK, gin.H{"error": "游标错误: " + err2.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": links})
}
