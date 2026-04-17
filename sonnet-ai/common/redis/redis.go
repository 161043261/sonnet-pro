package redis

import (
	"context"
	"fmt"
	"lark_ai/config"
	"strconv"
	"strings"

	redisCli "github.com/redis/go-redis/v9"
)

var Rdb *redisCli.Client

var ctx = context.Background()

func Init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.RedisHost
	port := conf.RedisConfig.RedisPort
	password := conf.RedisConfig.RedisPassword
	db := conf.RedisDb
	addr := host + ":" + strconv.Itoa(port)

	Rdb = redisCli.NewClient(&redisCli.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: 2, // Use Protocol 2 to avoid maint_notifications warning
	})

}

// InitRedisIndex initializes Redis index, supports differentiating by filename
func InitRedisIndex(ctx context.Context, filename string, dimension int) error {
	indexName := GenerateIndexName(filename)

	// Check if index exists
	_, err := Rdb.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil {
		fmt.Println("Index already exists, skipping creation")
		return nil
	}

	// Create new index if it does not exist
	if !strings.Contains(err.Error(), "Unknown index name") {
		return fmt.Errorf("failed to check index: %w", err)
	}

	fmt.Println("Creating Redis index...")

	prefix := GenerateIndexNamePrefix(filename)

	// Create index
	createArgs := []any{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err := Rdb.Do(ctx, createArgs...).Err(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	fmt.Println("Index created successfully!")
	return nil
}

// DeleteRedisIndex deletes Redis index, supports differentiating by filename
func DeleteRedisIndex(ctx context.Context, filename string) error {
	indexName := GenerateIndexName(filename)

	// Delete index
	if err := Rdb.Do(ctx, "FT.DROPINDEX", indexName).Err(); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	fmt.Println("Index deleted successfully!")
	return nil
}
