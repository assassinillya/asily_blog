package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetTags 获取所有标签列表
func GetTags(c *gin.Context) {
	db := utils.GetCollection("tags")

	opts := options.Find().SetSort(bson.D{{"count", -1}})

	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询标签失败: "+err.Error())
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var tags []models.Tag
	if err = cursor.All(context.Background(), &tags); err != nil {
		respondError(c, http.StatusInternalServerError, "解码标签数据失败: "+err.Error())
		return
	}

	if tags == nil {
		tags = []models.Tag{}
	}

	respondList(c, tags, nil, gin.H{
		"total": len(tags),
	})
}
