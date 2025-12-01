package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient redis.UniversalClient

func InitConnection() {
	if !AppConfigData.UseRedis {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%v", RedisConfigData.Host, RedisConfigData.Port),
			Username: "default",
			Password: RedisConfigData.Password,
			DB:       RedisConfigData.DB,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MaxRetries:   2,
		})
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			fmt.Println("REDIS ERROR:", err.Error())
			panic("cannot connect to Redis")
		}
	} else {
		// Create Redis Cluster Client
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    RedisClusterConfigData.Hosts,
			Password: RedisClusterConfigData.Password,
		})

		// Test Connection
		for _, addr := range RedisClusterConfigData.Hosts {
			nodeClient := redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: RedisClusterConfigData.Password,
			})

			_, err := nodeClient.Ping(context.Background()).Result()
			if err != nil {
				fmt.Println("REDIS CLUSTER ERROR:", err.Error())
				panic("Cannot connect to Redis Cluster")
			}

			nodeClient.Close()
		}
	}
}

func GetClient() redis.UniversalClient {
	return redisClient
}
