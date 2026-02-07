package sessionmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RedisSessionRulesStore manages session rules in Redis
type RedisSessionRulesStore struct {
	client *redis.Client
	logger zerolog.Logger
}

// NewRedisSessionRulesStore creates a new Redis session rules store
func NewRedisSessionRulesStore(client *redis.Client) *RedisSessionRulesStore {
	return &RedisSessionRulesStore{
		client: client,
		logger: log.With().Str("component", "sessionrules_store").Logger(),
	}
}

// StoreSessionRulesConfig stores the session rules config in Redis
func (s *RedisSessionRulesStore) StoreSessionRulesConfig(ctx context.Context, config *SessionRulesConfig) error {
	for sessionType, typeConfig := range config.SessionTypes {
		if !typeConfig.Enabled {
			continue
		}

		key := fmt.Sprintf("session:rules:%s", sessionType)

		// Marshal the type config to JSON
		data, err := json.Marshal(typeConfig)
		if err != nil {
			s.logger.Error().Err(err).Str("type", sessionType).Msg("Failed to marshal session type config")
			continue
		}

		// Store in Redis with TTL if configured
		ttl := time.Duration(0)
		if typeConfig.Redis.TTLSeconds > 0 {
			ttl = time.Duration(typeConfig.Redis.TTLSeconds) * time.Second
		}

		if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
			s.logger.Error().Err(err).Str("type", sessionType).Msg("Failed to store session type config in Redis")
			return err
		}

		s.logger.Info().Str("type", sessionType).Int("kpis", len(typeConfig.KPIs)).Msg("Session type config stored in Redis")
	}

	// Store global config as well
	globalKey := "session:config:global"
	globalData, err := json.Marshal(config.Global)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	if err := s.client.Set(ctx, globalKey, globalData, 0).Err(); err != nil {
		return fmt.Errorf("failed to store global config: %w", err)
	}

	return nil
}

// GetSessionTypeConfig retrieves a session type config from Redis
func (s *RedisSessionRulesStore) GetSessionTypeConfig(ctx context.Context, sessionType string) (*SessionTypeConfig, error) {
	key := fmt.Sprintf("session:rules:%s", sessionType)

	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session type '%s' not found in Redis", sessionType)
		}
		return nil, fmt.Errorf("failed to retrieve session type config: %w", err)
	}

	config := &SessionTypeConfig{}
	if err := json.Unmarshal([]byte(data), config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session type config: %w", err)
	}

	return config, nil
}

// GetAllSessionTypes retrieves all enabled session types from Redis
func (s *RedisSessionRulesStore) GetAllSessionTypes(ctx context.Context) (map[string]*SessionTypeConfig, error) {
	keys := []string{}
	pattern := "session:rules:*"

	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan session rules keys: %w", err)
	}

	result := make(map[string]*SessionTypeConfig)

	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Result()
		if err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("Failed to retrieve session type config")
			continue
		}

		config := &SessionTypeConfig{}
		if err := json.Unmarshal([]byte(data), config); err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("Failed to unmarshal session type config")
			continue
		}

		// Extract session type name from key (session:rules:{type})
		var sessionType string
		fmt.Sscanf(key, "session:rules:%s", &sessionType)
		result[sessionType] = config
	}

	return result, nil
}

// DeleteSessionRulesConfig removes session rules from Redis
func (s *RedisSessionRulesStore) DeleteSessionRulesConfig(ctx context.Context, sessionType string) error {
	key := fmt.Sprintf("session:rules:%s", sessionType)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session rules for type '%s': %w", sessionType, err)
	}
	return nil
}

// SubscribeToSessionUpdates subscribes to session update notifications
func (s *RedisSessionRulesStore) SubscribeToSessionUpdates(ctx context.Context, sessionType string) <-chan *redis.Message {
	channel := fmt.Sprintf("session_updates:%s", sessionType)
	return s.client.Subscribe(ctx, channel).Channel()
}

// PublishSessionUpdate publishes a session update notification
func (s *RedisSessionRulesStore) PublishSessionUpdate(ctx context.Context, sessionType string, message interface{}) error {
	channel := fmt.Sprintf("session_updates:%s", sessionType)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, channel, string(data)).Err()
}

// HealthCheck verifies Redis connection and session rules existence
func (s *RedisSessionRulesStore) HealthCheck(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	// Check if at least one session type config exists
	count, err := s.client.DBSize(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to check redis size: %w", err)
	}

	if count == 0 {
		s.logger.Warn().Msg("No session rules found in Redis")
	}

	return nil
}
