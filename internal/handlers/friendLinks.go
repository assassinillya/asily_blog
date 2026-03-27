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

type friendLinkPayload struct {
	ID          string `json:"_id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}

type friendLinkResponse struct {
	ID          string `json:"_id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}

func buildFriendLinkResponse(link models.FriendLink) friendLinkResponse {
	return friendLinkResponse{
		ID:          link.ID.Hex(),
		Name:        link.Name,
		URL:         link.URL,
		Description: link.Description,
		Avatar:      link.Avatar,
	}
}

func validateFriendLinkPayload(link friendLinkPayload) string {
	if link.Name == "" {
		return "友情链接名称不能为空"
	}
	if link.URL == "" {
		return "友情链接地址不能为空"
	}

	return ""
}

// AddLink 添加一个新的友情链接
func AddLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data friendLinkPayload
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	if msg := validateFriendLinkPayload(data); msg != "" {
		respondError(c, http.StatusBadRequest, msg)
		return
	}

	link := models.FriendLink{
		Name:        data.Name,
		URL:         data.URL,
		Description: data.Description,
		Avatar:      data.Avatar,
	}

	result, err := db.InsertOne(context.Background(), link)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "友情链接插入失败: "+err.Error())
		return
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		respondError(c, http.StatusInternalServerError, "友情链接插入成功，但返回ID解析失败")
		return
	}

	link.ID = insertedID
	respondMessageData(c, http.StatusCreated, "友情链接添加成功", buildFriendLinkResponse(link))
}

// DeleteLink 删除一个友情链接
func DeleteLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的友链ID格式")
		return
	}

	result, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "友情链接删除失败: "+err.Error())
		return
	}

	if result.DeletedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要删除的友情链接")
		return
	}

	respondMessage(c, http.StatusOK, "友情链接删除成功")
}

// UpdateLink 更新一个友情链接
func UpdateLink(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

	var data friendLinkPayload
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	if data.ID == "" {
		respondError(c, http.StatusBadRequest, "友情链接ID不能为空")
		return
	}
	if msg := validateFriendLinkPayload(data); msg != "" {
		respondError(c, http.StatusBadRequest, msg)
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的友链ID格式")
		return
	}

	update := bson.M{
		"$set": bson.M{
			"name":        data.Name,
			"url":         data.URL,
			"description": data.Description,
			"avatar":      data.Avatar,
		},
	}

	result, err := db.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "友情链接更新失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要更新的友情链接")
		return
	}

	respondMessageData(c, http.StatusOK, "友情链接更新成功", friendLinkResponse{
		ID:          data.ID,
		Name:        data.Name,
		URL:         data.URL,
		Description: data.Description,
		Avatar:      data.Avatar,
	})
}

// GetLinks 获取友情链接列表（支持分页）
func GetLinks(c *gin.Context) {
	db := utils.GetCollection("friendLinks")

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
	opts := options.Find().
		SetSort(bson.D{{"_id", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询友情链接失败: "+err.Error())
		return
	}
	defer func() {
		if closeErr := cursor.Close(context.Background()); closeErr != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", closeErr)
		}
	}()

	var links []models.FriendLink
	if err = cursor.All(context.Background(), &links); err != nil {
		respondError(c, http.StatusInternalServerError, "解码友情链接数据失败: "+err.Error())
		return
	}

	list := make([]friendLinkResponse, 0, len(links))
	for _, link := range links {
		list = append(list, buildFriendLinkResponse(link))
	}

	total, err := db.CountDocuments(context.Background(), bson.D{{}})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询友情链接总数失败: "+err.Error())
		return
	}

	respondList(c, list, &pagination{
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil)
}
