package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB MongoDB连接管理器
type MongoDB struct {
	client *mongo.Client
	dbName string
}

// NewMongoDB 创建MongoDB连接
func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &MongoDB{
		client: client,
		dbName: dbName,
	}, nil
}

// GetClient 获取MongoDB客户端
func (m *MongoDB) GetClient() *mongo.Client {
	return m.client
}

// GetDatabase 获取数据库
func (m *MongoDB) GetDatabase() *mongo.Database {
	return m.client.Database(m.dbName)
}

// GetCollection 获取集合
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.client.Database(m.dbName).Collection(name)
}

// Close 关闭连接
func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
} 