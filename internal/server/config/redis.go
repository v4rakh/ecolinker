package config

import (
	"context"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a new Redis client connection
func NewRedisClient(name string, url string) (*redis.Client, error) {
	if name == "" || url == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var redisOptions *redis.Options
	if redisOptions, err = redis.ParseURL(url); err != nil {
		return nil, fmt.Errorf("cannot parse REDIS URL '%s': %s", url, err)
	}

	redisOptions.ClientName = name

	c := redis.NewClient(redisOptions)

	if err = c.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to REDIS: %w", err)
	}

	return c, nil
}
