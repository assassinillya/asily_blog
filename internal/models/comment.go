package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Comment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BlogID    primitive.ObjectID `bson:"blogId" json:"blogId"` // 关联的博客ID
	Username  string             `bson:"username" json:"username"`
	QQ        string             `bson:"qq" json:"qq"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
