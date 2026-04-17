package main

import (
	"log"
	"lark_ai/common/ai_agent"
	"lark_ai/common/kafka"
	"lark_ai/common/mysql"
	"lark_ai/common/redis"
	"lark_ai/config"
	"lark_ai/dao/message"
	"lark_ai/router"
)

func StartServer(addr string, port int) error {
	r := router.InitRouter(addr, port)
	// Static resource path mapping for the server, not needed currently
	// r.Static(config.GetConfig().HttpFilePath, config.GetConfig().MusicFilePath)
	r.Spin()
	return nil
}

// Load messages from database and initialize AIAgentManager
func readDataFromDB() error {
	manager := ai_agent.GetGlobalManager()
	// Read all messages from database
	msgs, err := message.GetAllMessages()
	if err != nil {
		return err
	}
	// Iterate through database messages
	for i := range msgs {
		m := &msgs[i]
		modelType := "ollama"
		config := make(map[string]any)

		// Create corresponding AIAgent
		helper, err := manager.GetOrCreateAIAgent(m.UserName, m.SessionID, modelType, config)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		log.Println("readDataFromDB init:  ", helper.SessionID)
		// Add message to memory (storage disabled)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	log.Println("AIAgentManager init success ")
	return nil
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	if err := mysql.InitMysql(); err != nil {
		log.Println("InitMysql error , " + err.Error())
		return
	}
	// Initialize AIAgentManager
	readDataFromDB()

	// Initialize Redis
	redis.Init()
	log.Println("redis init success  ")
	kafka.InitKafka(conf.KafkaConfig.KafkaBrokers, conf.KafkaConfig.KafkaTopic)
	log.Println("kafka init success  ")

	err := StartServer(host, port) // Start HTTP server
	if err != nil {
		panic(err)
	}
}
