package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type KafkaProducerConfig struct {
	BrokerURL    string
	Topic        string
	QueueSize    int
	WriteTimeout time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
}

func (c *KafkaProducerConfig) applyDefaults() {
	if c.QueueSize <= 0 {
		c.QueueSize = 100
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 5 * time.Second
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = 500 * time.Millisecond
	}
}

type KafkaProducer struct {
	writer    *kafkago.Writer
	topic     string
	cfg       KafkaProducerConfig
	events    chan SiteCheckEvent
	done      chan struct{}
	closeOnce sync.Once
	stopped   chan struct{}
}

func NewKafkaProducer(cfg KafkaProducerConfig) *KafkaProducer {
	cfg.applyDefaults()

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.BrokerURL),
		Topic:        cfg.Topic,
		Balancer:     &kafkago.Hash{},
		RequiredAcks: kafkago.RequireOne,
	}

	p := &KafkaProducer{
		writer:  writer,
		topic:   cfg.Topic,
		cfg:     cfg,
		events:  make(chan SiteCheckEvent, cfg.QueueSize),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go p.run()
	return p
}

func (p *KafkaProducer) Publish(_ context.Context, event SiteCheckEvent) error {
	select {
	case p.events <- event:
		return nil
	default:
		slog.Warn("kafka queue is full, dropping event", slog.String("site_id", event.SiteID))
		return fmt.Errorf("publish queue full")
	}
}

func (p *KafkaProducer) run() {
	defer close(p.stopped)

	for {
		select {
		case e := <-p.events:
			p.writeWithRetry(e)
		case <-p.done:
			for {
				select {
				case e := <-p.events:
					p.writeWithRetry(e)
				default:
					return
				}
			}
		}
	}
}

func (p *KafkaProducer) writeWithRetry(event SiteCheckEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", slog.String("event_id", event.SiteID), slog.String("error", err.Error()))
		return
	}

	msg := kafkago.Message{
		Key:   []byte(event.SiteID),
		Value: payload,
		Time:  event.CheckedAt,
	}

	var lastErr error
	for att := 1; att <= p.cfg.MaxRetries; att++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.cfg.WriteTimeout)
		err := p.writer.WriteMessages(ctx, msg)
		cancel()

		if err == nil {
			slog.Info("event published", slog.String("site_id", event.SiteID), slog.String("topic", p.cfg.Topic))
			return
		}

		lastErr = err
		slog.Warn("failed to publish event, retrying",
			slog.String("site_id", event.SiteID),
			slog.Int("attempt", att),
			slog.String("error", err.Error()),
		)
		time.Sleep(p.cfg.RetryBackoff * time.Duration(att))
	}

	if lastErr != nil {
		slog.Error("failed to publish event after retries",
			slog.String("site_id", event.SiteID),
			slog.String("error", lastErr.Error()),
		)
	}
}

func (p *KafkaProducer) Close() error {
	var err error
	p.closeOnce.Do(func() {
		close(p.done)
		<-p.stopped
		err = p.writer.Close()
	})
	return err
}
