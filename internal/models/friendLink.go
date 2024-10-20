package models

type FriendLink struct {
	ID          string `bson:"_id,omitempty" json:"_id"`
	Name        string `bson:"name" json:"name"`
	URL         string `bson:"url" json:"url"`
	Description string `bson:"description" json:"description"`
	Avatar      string `bson:"avatar" json:"avatar"`
}
