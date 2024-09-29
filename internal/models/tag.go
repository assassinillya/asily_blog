package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Tag struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Count int                `bson:"count" json:"count"` // 出现次数
}
