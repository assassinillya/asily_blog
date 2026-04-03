package utils

import (
	"asily_blog/pkg/config"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var client *mongo.Client

// ConnectDB 连接到 MongoDB 数据库
func ConnectDB() {
	var err error
	clientOptions := options.Client().ApplyURI("mongodb://" + config.C.MongoDB)

	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	if err = ensureIndexes(context.TODO()); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to MongoDB!")
}

// GetCollection 获取db集合
func GetCollection(collectionName string) *mongo.Collection {
	return client.Database("asily_blog").Collection(collectionName)
}

func ensureIndexes(ctx context.Context) error {
	comments := GetCollection("comments")
	_, err := comments.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "blogId", Value: 1},
				{Key: "parentId", Value: 1},
				{Key: "createdAt", Value: 1},
			},
			Options: options.Index().SetName("idx_blog_parent_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "rootId", Value: 1},
				{Key: "createdAt", Value: 1},
			},
			Options: options.Index().SetName("idx_root_created_at"),
		},
	})
	return err
}
