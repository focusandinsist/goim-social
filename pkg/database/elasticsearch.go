package database

import (
	"context"
	"fmt"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"goim-social/pkg/logger"
)

// ElasticSearch ElasticSearch客户端封装
type ElasticSearch struct {
	client *elasticsearch.Client
	logger logger.Logger
}

// NewElasticSearch 创建ElasticSearch连接
func NewElasticSearch(logger logger.Logger) (*ElasticSearch, error) {
	// 使用默认配置
	addresses := []string{"http://localhost:9200"}
	
	// 从环境变量获取配置
	if esAddr := os.Getenv("ELASTICSEARCH_URL"); esAddr != "" {
		addresses = []string{esAddr}
	}
	
	username := os.Getenv("ELASTICSEARCH_USERNAME")
	password := os.Getenv("ELASTICSEARCH_PASSWORD")

	esConfig := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}

	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ElasticSearch client: %v", err)
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ElasticSearch: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ElasticSearch connection error: %s", res.String())
	}

	logger.Info(context.Background(), "ElasticSearch connected successfully")
	
	return &ElasticSearch{
		client: client,
		logger: logger,
	}, nil
}

// GetClient 获取原生客户端
func (es *ElasticSearch) GetClient() *elasticsearch.Client {
	return es.client
}

// Close 关闭连接
func (es *ElasticSearch) Close() error {
	// ElasticSearch客户端不需要显式关闭
	return nil
}

// Ping 测试连接
func (es *ElasticSearch) Ping(ctx context.Context) error {
	res, err := es.client.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ElasticSearch ping failed: %s", res.String())
	}

	return nil
}
