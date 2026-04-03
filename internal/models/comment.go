package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Comment struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	BlogID          primitive.ObjectID  `bson:"blogId" json:"blogId"` // 关联的博客ID
	ParentID        *primitive.ObjectID `bson:"parentId,omitempty" json:"parentId,omitempty"`
	RootID          *primitive.ObjectID `bson:"rootId,omitempty" json:"rootId,omitempty"` // 顶层楼层评论ID
	ReplyToUsername string              `bson:"replyToUsername,omitempty" json:"replyToUsername,omitempty"`
	Username        string              `bson:"username" json:"username"`
	QQ              string              `bson:"qq" json:"qq"`
	Content         string              `bson:"content" json:"content"`
	CreatedAt       time.Time           `bson:"createdAt" json:"createdAt"`
	Like            int                 `json:"like" bson:"like"`
	LikeCount       int                 `json:"likeCount" bson:"likeCount"`
	UnLikeCount     int                 `json:"unlikeCount" bson:"unlikeCount"`
}
