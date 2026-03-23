package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddLink 添加一个新的友情链接
func AddLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")
	var link models.FriendLink

	// 绑定并验证请求数据
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 校验必填字段
	if link.Name == "" || link.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "链接名称和URL是必填项"})
		return
	}

	// 插入新的友情链接
	_, err := db.InsertOne(context.Background(), link)
	if err != nil {
		// 状态码修正：数据库插入失败是服务器内部错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "友情链接插入失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "友情链接添加成功"}) // 状态码改进：使用 201 Created 更合适
}

// DeleteLink 删除一个友情链接
func DeleteLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data struct {
		ID string `json:"_id"` // 修正：字段名应为大写 ID 以匹配 Go 的导出规则
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查ID格式是否有效
	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID格式"})
		return
	}

	// 优化：直接执行删除操作，并检查结果，无需预先查询
	result, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "友情链接删除失败: " + err.Error()})
		return
	}

	// 检查是否真的删除了一个文档
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到要删除的友情链接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "友情链接删除成功"})
}

// UpdateLink 更新一个友情链接
func UpdateLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var link models.FriendLink
	// 绑定请求中的 JSON 数据
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// BUG修复：检查ID格式是否有效
	id, err := primitive.ObjectIDFromHex(link.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID格式"})
		return
	}

	// 定义要更新的字段
	update := bson.M{
		"$set": bson.M{
			"name":        link.Name,
			"avatar":      link.Avatar,
			"url":         link.URL,
			"description": link.Description,
		},
	}

	// 优化：直接执行更新操作，并检查结果
	result, err := db.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "友情链接更新失败: " + err.Error()})
		return
	}

	// 检查是否真的找到了并更新了文档
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到要更新的友情链接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "友情链接更新成功"})
}

// GetLinks 获取友情链接列表（支持分页）
func GetLinks(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	// 从 URL query 参数获取分页信息，并提供默认值
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	// 变量名修正：使用独立的变量名以提高可读性
	page, pageErr := strconv.Atoi(pageStr)
	if pageErr != nil || page < 1 {
		page = 1 // 如果参数无效，使用安全的默认值
	}

	limit, limitErr := strconv.Atoi(limitStr)
	if limitErr != nil || limit < 1 {
		limit = 10 // 如果参数无效，使用安全的默认值
	}

	skip := (page - 1) * limit

	// 设置查询选项
	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}
	// 改进：在 defer 中检查 Close() 的错误
	defer func() {
		if closeErr := cursor.Close(context.Background()); closeErr != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", closeErr)
		}
	}()

	var links []models.FriendLink
	// 使用 cursor.All() 一次性解码所有结果
	if err = cursor.All(context.Background(), &links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解码失败: " + err.Error()})
		return
	}

	// 健壮性改进：如果结果为 nil，返回一个空数组
	if links == nil {
		links = []models.FriendLink{}
	}

	// 同时返回总数，便于前端分页
	total, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询友链总数失败: " + err.Error()})
		return
	}

	// API响应结构优化
	c.JSON(http.StatusOK, gin.H{"links": links, "total": total})
}
