package utils

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var client *mongo.Client

// ConnectDB 连接到 MongoDB 数据库
func ConnectDB() {
	var err error
	clientOptions := options.Client().ApplyURI("mongodb://120.78.234.30:27017")

	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to MongoDB!")
}

// GetCollection 获取db集合
func GetCollection(collectionName string) *mongo.Collection {
	return client.Database("asily_blog").Collection(collectionName)
}
