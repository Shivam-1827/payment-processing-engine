package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	PublishIntent(ctx context.Context, key string, payload any) error
	Close() error
}

type KafkaProducer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewKafkaProducer(brokers []string, topic string, logger *slog.Logger) *KafkaProducer {
	w := &kafka.Writer{
		Addr:			kafka.TCP(brokers...),
		Topic:			topic,
		Balancer: 		&kafka.Hash{},
		RequiredAcks: 	kafka.RequireAll,
		MaxAttempts: 	5,
		BatchTimeout: 	10* time.Millisecond,
		AllowAutoTopicCreation: true,
	}

	return  &KafkaProducer{writer: w, logger: logger}
}

func (k *KafkaProducer) PublishIntent(ctx context.Context, key string, payload any) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := kafka.Message{
		Key: []byte(key),
		Value: bytes,
		Time: time.Now(),
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := k.writer.WriteMessages(publishCtx, msg); err != nil {
		k.logger.Error("failed to publish to kafka", "key", key, "error", err)
		return fmt.Errorf("kafka write failed: %w", err)
	}

	return nil
}

func (k *KafkaProducer) Close() error {
	return k.writer.Close()
}