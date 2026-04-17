package config

import (
	"encoding/json"
	"log"
	"os"
)

type MainConfig struct {
	Port    int    `json:"port"`
	AppName string `json:"app_name"`
	Host    string `json:"host"`
}

type RedisConfig struct {
	RedisPort     int    `json:"port"`
	RedisDb       int    `json:"db"`
	RedisHost     string `json:"host"`
	RedisPassword string `json:"password"`
}

type MysqlConfig struct {
	MysqlPort         int    `json:"port"`
	MysqlHost         string `json:"host"`
	MysqlUser         string `json:"user"`
	MysqlPassword     string `json:"password"`
	MysqlDatabaseName string `json:"database_name"`
	MysqlCharset      string `json:"charset"`
}

type JwtConfig struct {
	ExpireDuration int    `json:"expire_duration"`
	Issuer         string `json:"issuer"`
	Subject        string `json:"subject"`
	Key            string `json:"key"`
}

type KafkaConfig struct {
	KafkaBrokers []string `json:"brokers"`
	KafkaTopic   string   `json:"topic"`
}

type Config struct {
	RedisConfig  `json:"redis_config"`
	MysqlConfig  `json:"mysql_config"`
	JwtConfig    `json:"jwt_config"`
	MainConfig   `json:"main_config"`
	KafkaConfig  `json:"kafka_config"`
	OllamaConfig `json:"ollama_config"`
}

type OllamaConfig struct {
	BaseURL        string `json:"base_url"`
	ModelName      string `json:"model_name"`
	EmbeddingModel string `json:"embedding_model"`
	Dimension      int    `json:"dimension"`
	DocDir         string `json:"docs_dir"`
}

type RedisKeyConfig struct {
	IndexName       string
	IndexNamePrefix string
}

var DefaultRedisKeyConfig = RedisKeyConfig{
	IndexName:       "rag_docs:%s:idx",
	IndexNamePrefix: "rag_docs:%s:",
}

var config *Config

// InitConfig initializes project configuration
func InitConfig() error {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	if err := json.Unmarshal(data, config); err != nil {
		log.Fatal(err.Error())
		return err
	}
	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		_ = InitConfig()
	}
	return config
}
