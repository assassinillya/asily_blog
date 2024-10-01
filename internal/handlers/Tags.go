package handlers

import (
	"asily_blog/internal/utils"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

func Tags(c *gin.Context) {
	db := utils.GetCollection("tags")
	var tags []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	Find, err := db.Find(context.Background(), bson.D{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "查询错误 " + err.Error()})
		return
	}
	defer Find.Close(context.Background())

	for Find.Next(context.Background()) {
		var tag struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		if err1 := Find.Decode(&tag); err1 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "转码失败 " + err1.Error()})
			return
		}

		tags = append(tags, tag)
	}

	c.JSON(http.StatusOK, gin.H{"message": tags})
}
