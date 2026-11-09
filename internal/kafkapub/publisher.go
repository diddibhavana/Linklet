package kafkapub

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Publisher struct { writer *kafka.Writer }

func New(brokers []string) *Publisher {
	return &Publisher{writer:&kafka.Writer{Addr:kafka.TCP(brokers...),Balancer:&kafka.Hash{},RequiredAcks:kafka.RequireAll,Async:false,WriteTimeout:10*time.Second}}
}
func (p *Publisher) Publish(ctx context.Context, topic,key string,payload []byte) error {
	return p.writer.WriteMessages(ctx,kafka.Message{Topic:topic,Key:[]byte(key),Value:payload})
}
func (p *Publisher) Close() error { return p.writer.Close() }
