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

	// 定义查询选项，按标签引用次数降序排序
	opts := options.Find().SetSort(bson.D{{"count", -1}})

	// 执行查询
	cursor, err := db.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		// 状态码修正：数据库查询失败是服务器内部错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询标签失败: " + err.Error()})
		return
	}
	// 改进：在 defer 中检查 Close() 的错误
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			// 在生产环境中，这里应该使用日志库记录错误
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	// 结构体复用：使用 models.Tag 代替匿名结构体
	var tags []models.Tag

	// 使用 cursor.All() 一次性解码所有结果，代码更简洁
	if err = cursor.All(context.Background(), &tags); err != nil {
		// 状态码修正：解码失败是服务器内部错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解码标签数据失败: " + err.Error()})
		return
	}

	// 健壮性改进：如果数据库中没有标签，返回一个空数组而不是 null
	if tags == nil {
		tags = []models.Tag{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(tags),
		"list":  tags,
	})
}
