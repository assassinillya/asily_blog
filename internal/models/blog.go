package models

import (
	"time"
)

type Blog struct {
	ID        string    `bson:"_id,omitempty" json:"_id"`
	Title     string    `bson:"title" json:"title"`
	Content   string    `bson:"content" json:"content"` // Markdown格式
	Tags      []string  `bson:"tags" json:"tags"`       // 标签数组
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	Views     int       `bson:"views" json:"views"` // 阅读量
}
