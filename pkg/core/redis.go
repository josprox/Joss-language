package core

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	GlobalRedis *redis.Client
	Ctx         = context.Background()
)

// InitRedis initializes the global Redis client
func InitRedis(addr, password string, db int) {
	GlobalRedis = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// Redis Native Class
func (r *Runtime) executeRedisMethod(instance *Instance, method string, args []interface{}) interface{} {
	if method == "connect" {
		// Manual connection: Redis.connect("localhost:6379", "", 0)
		if len(args) >= 1 {
			addr := args[0].(string)
			password := ""
			db := 0
			if len(args) > 1 {
				password = args[1].(string)
			}
			if len(args) > 2 {
				if d, ok := args[2].(int); ok {
					db = d
				} else if d, ok := args[2].(float64); ok {
					db = int(d)
				}
			}
			InitRedis(addr, password, db)
			return true
		}
		return false
	}

	if GlobalRedis == nil {
		fmt.Println("[Redis] Error: Not connected (Call Redis.connect first or configure .env)")
		return nil
	}

	switch method {
	case "set":
		// Redis.set("key", "value", seconds_ttl)
		if len(args) >= 2 {
			key := args[0].(string)
			val := args[1]
			ttl := time.Duration(0)
			if len(args) > 2 {
				if t, ok := args[2].(int); ok {
					ttl = time.Duration(t) * time.Second
				} else if t, ok := args[2].(float64); ok {
					ttl = time.Duration(int(t)) * time.Second
				}
			}
			err := GlobalRedis.Set(Ctx, key, val, ttl).Err()
			if err != nil {
				fmt.Printf("[Redis] Set error: %v\n", err)
				return false
			}
			return true
		}

	case "get":
		// Redis.get("key")
		if len(args) >= 1 {
			key := args[0].(string)
			val, err := GlobalRedis.Get(Ctx, key).Result()
			if err == redis.Nil {
				return nil
			} else if err != nil {
				fmt.Printf("[Redis] Get error: %v\n", err)
				return nil
			}
			return val
		}

	case "del":
		// Redis.del("key")
		if len(args) >= 1 {
			key := args[0].(string)
			err := GlobalRedis.Del(Ctx, key).Err()
			if err != nil {
				return false
			}
			return true
		}
	}
	return nil
}
