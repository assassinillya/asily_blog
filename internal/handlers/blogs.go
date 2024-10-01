package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

// InsertBlog 插入文档
func InsertBlog(c *gin.Context) {

	db := utils.GetCollection("blogs")
	var blog models.Blog
	if err := c.ShouldBindJSON(&blog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	blog.UpdatedAt = time.Now()
	blog.CreatedAt = time.Now()
	blog.Views = 0
	data, err := bson.Marshal(blog)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理Tags
	tagsDB := utils.GetCollection("tags")
	for _, tag := range blog.Tags {
		ok := tagsDB.FindOne(context.Background(), bson.M{"name": tag})
		if ok.Err() != nil {
			// 找不到这个tag
			_, err = tagsDB.InsertOne(context.Background(), bson.M{
				"name":  tag,
				"count": 1,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		} else {
			_, err = tagsDB.UpdateOne(context.Background(), bson.M{
				"name": tag,
			}, bson.M{
				"$inc": bson.M{
					"count": 1,
				},
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	result, err1 := db.InsertOne(context.Background(), data)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "插入成功",
		"_id":     result.InsertedID,
	})
}

// ViewAdd 阅读量增加 疑似写了没用
func ViewAdd(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := primitive.ObjectIDFromHex(data.Id)
	//fmt.Println(data.Id)
	update := bson.M{
		"$inc": bson.M{"views": 1},
	}

	_, err := db.UpdateByID(context.Background(), id, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败, 原因:" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ReSetBlog 编辑文档
func ReSetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data models.Blog
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := primitive.ObjectIDFromHex(data.ID)
	fmt.Println(id)
	fmt.Println(data.Tags)
	// 处理tags
	// 减去原tags, 增加新tags
	tagsDB := utils.GetCollection("tags")
	// 减去原tag
	// 先根据_id查询原博客的tags数组

	var oldBlog models.Blog
	ok1 := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&oldBlog)
	if ok1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "找不到原博客"})
		return
	}

	for _, tag := range oldBlog.Tags {
		ok := tagsDB.FindOne(context.Background(), bson.M{"name": tag})
		if ok.Err() != nil {
			// 找不到这个tag
			c.JSON(http.StatusBadRequest, gin.H{"error": "找不到原tag"})
			return
		}
		_, err := tagsDB.UpdateOne(context.Background(), bson.M{
			"name": tag,
		}, bson.M{
			"$inc": bson.M{
				"count": -1,
			},
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "删除原tag失败:=, 失败原因:" + err.Error()})
			return
		}
	}
	// 增加新tag
	for _, tag := range data.Tags {
		ok := tagsDB.FindOne(context.Background(), bson.M{"name": tag})
		if ok.Err() != nil {
			// 找不到这个tag
			_, err := tagsDB.InsertOne(context.Background(), bson.M{
				"name":  tag,
				"count": 1,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		} else {
			_, err := tagsDB.UpdateOne(context.Background(), bson.M{
				"name": tag,
			}, bson.M{
				"$inc": bson.M{
					"count": 1,
				},
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	update := bson.M{
		"$set": bson.M{
			"title":     data.Title,
			"tags":      data.Tags,
			"content":   data.Content,
			"updatedAt": time.Now(),
		},
	}

	_, err := db.UpdateByID(context.Background(), id, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败, 原因:" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func GetBlog(c *gin.Context) {
	db := utils.GetCollection("blogs")
	var data struct {
		Id string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := primitive.ObjectIDFromHex(data.Id)

	var blog models.Blog

	ok := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&blog)
	blog.Views++ // 给当前的博客加上当前阅读量

	if ok != nil {
		c.JSON(http.StatusOK, gin.H{"err": "找不到此博客"})
		return
	}

	update := bson.M{
		"$inc": bson.M{"views": 1},
	}

	_, err := db.UpdateByID(context.Background(), id, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败, 原因:" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": blog})

}
