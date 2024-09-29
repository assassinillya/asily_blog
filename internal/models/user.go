package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username    string             `bson:"username" json:"username"`
	QQ          string             `bson:"qq" json:"qq"`
	Password    string             `bson:"password" json:"password"`
	Avatar      string             `bson:"avatar" json:"avatar"`
	Description string             `bson:"description" json:"description"`
}
