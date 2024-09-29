package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type FriendLink struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	URL         string             `bson:"url" json:"url"`
	Description string             `bson:"description" json:"description"`
}
