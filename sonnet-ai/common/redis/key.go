package redis

import (
	"fmt"
	"lark_ai/config"
)

func GenerateIndexName(filename string) string {
	indexName := fmt.Sprintf(config.DefaultRedisKeyConfig.IndexName, filename)
	return indexName
}

func GenerateIndexNamePrefix(filename string) string {
	prefix := fmt.Sprintf(config.DefaultRedisKeyConfig.IndexNamePrefix, filename)
	return prefix
}
