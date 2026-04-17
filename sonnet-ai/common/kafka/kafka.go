package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

var (
	GlobalKafkaClient *KafkaClient
)

func InitKafka(brokers []string, topic string) {
	// Initialize writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// Initialize reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  "lark_ai_group",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	GlobalKafkaClient = &KafkaClient{
		writer: writer,
		reader: reader,
	}

	// Start consuming
	go GlobalKafkaClient.Consume(context.Background())
}

func (k *KafkaClient) Publish(ctx context.Context, message []byte) error {
	err := k.writer.WriteMessages(ctx,
		kafka.Message{
			Value: message,
		},
	)
	if err != nil {
		log.Printf("failed to write messages: %v", err)
	}
	return err
}

func (k *KafkaClient) Consume(ctx context.Context) {
	for {
		m, err := k.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("could not read message: %v", err)
			continue
		}
		if err := MQMessage(m.Value); err != nil {
			log.Printf("failed to process message: %v", err)
		}
	}
}

func DestroyKafka() {
	if GlobalKafkaClient != nil {
		if err := GlobalKafkaClient.writer.Close(); err != nil {
			log.Printf("failed to close writer: %v", err)
		}
		if err := GlobalKafkaClient.reader.Close(); err != nil {
			log.Printf("failed to close reader: %v", err)
		}
	}
}
