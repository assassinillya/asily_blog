package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Test(c *gin.Context) {
	db := utils.GetCollection("blogs")
	//
	// 插入文档
	//blog := models.Blog{
	//	Title:     "测试博客",
	//	Content:   "测试插入博客",
	//	Tags:      []string{"golang", "mongodb"},
	//	CreatedAt: time.Now(),
	//	UpdatedAt: time.Now(),
	//	Views:     0,
	//}
	//_,err:=db.InsertOne(context.Background(),blog)
	//if err!=nil{
	//	log.Println("插入数据失败",err)
	//}

	// 查询文档
	//var blog1 models.Blog
	//err := db.FindOne(context.Background(), bson.M{"title":"测试博客"}).Decode(&blog1)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//fmt.Println(blog1.ID, blog1.Title, blog1.Content)
	//c.JSON(http.StatusOK, gin.H{"message": blog1})

	// 获取json并插入文档

	var blog models.Blog
	if err := c.ShouldBindJSON(blog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	_, err := db.InsertOne(context.Background(), blog)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

}
