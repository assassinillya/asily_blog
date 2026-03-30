package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"code":  status,
		"error": message,
	})
}

func respondData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"code": status,
		"data": data,
	})
}

func respondList(c *gin.Context, list any, page *pagination, extra gin.H) {
	body := gin.H{
		"code": http.StatusOK,
		"list": list,
	}
	if page != nil {
		body["pagination"] = page
	}
	for key, value := range extra {
		body[key] = value
	}

	c.JSON(http.StatusOK, body)
}

func respondMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"code":    status,
		"message": message,
	})
}

func respondMessageData(c *gin.Context, status int, message string, data any) {
	c.JSON(status, gin.H{
		"code":    status,
		"message": message,
		"data":    data,
	})
}
