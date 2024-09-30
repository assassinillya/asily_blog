package utils

import (
	"asily_blog/pkg/config"
	"context"
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

	log.Println("Connected to MongoDB!")
}

// GetCollection 获取db集合
func GetCollection(collectionName string) *mongo.Collection {
	return client.Database("asily_blog").Collection(collectionName)
}
