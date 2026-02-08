package nats

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// Config تنظیمات اتصال به NATS
type Config struct {
	URL            string        `json:"url"`
	ReconnectWait  time.Duration `json:"reconnect_wait"`
	MaxReconnects  int           `json:"max_reconnects"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
}

// Client کلاینت NATS با قابلیت‌های concurrency-safe
type Client struct {
	nc     *nats.Conn
	config Config
	mu     sync.Mutex
	closed bool
}

// NewClient ساخت یک کلاینت جدید NATS
func NewClient(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		cfg.URL = "nats://localhost:4222"
	}
	if cfg.ReconnectWait == 0 {
		cfg.ReconnectWait = 2 * time.Second
	}
	if cfg.MaxReconnects == 0 {
		cfg.MaxReconnects = -1
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}

	opts := []nats.Option{
		nats.Name("signalblocks-client"),
		nats.Timeout(cfg.ConnectTimeout),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Warn().Err(err).Msg("NATS disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Info().Msg("NATS reconnected")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			log.Error().Err(err).Msg("NATS error")
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	log.Info().Str("url", cfg.URL).Msg("Connected to NATS")

	return &Client{
		nc:     nc,
		config: cfg,
	}, nil
}

// Publish ارسال پیام به یک subject
func (c *Client) Publish(subject string, data []byte) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	return c.nc.Publish(subject, data)
}

// Subscribe ساده
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	return c.nc.Subscribe(subject, handler)
}

// Close بستن کلاینت
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	err := c.nc.Drain()
	if err != nil {
		log.Error().Err(err).Msg("Failed to drain NATS connection")
	}
	c.nc.Close()
	log.Info().Msg("NATS client closed")
	return nil
}