package service

import (
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
)

var redisClient *redis.Client = nil

type BackendStoreServiceType uint

const (
	Redis BackendStoreServiceType = iota
	Memcached
)

var DefaultBackendStore BackendStoreServiceType = Redis

type BackendStoreService struct{}

func init() {

	sessionBackend := strings.ToLower(utility.GetEnv("SESSION_BACKEND", "cookie"))

	// redis based
	if sessionBackend == "redis" {
		//Initializing redis
		dsn := utility.GetEnv("SESSION_BACKEND_HOST", "localhost:6379")
		redisClient = redis.NewClient(&redis.Options{
			Addr: dsn, //redis port
		})
		_, err := redisClient.Ping().Result()
		if err != nil {
			redisClient = nil
		}
		DefaultBackendStore = Redis
	} else if sessionBackend == "memcached" {
		DefaultBackendStore = Memcached
	}

}

func (b *BackendStoreService) IsServiceAvailable() bool {

	if DefaultBackendStore == Redis {
		return redisClient != nil
	}

	return false
}

func (b *BackendStoreService) Set(key, value string, expire_in time.Duration) (error error) {

	error = nil

	if !b.IsServiceAvailable() {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	} else if DefaultBackendStore == Redis {
		if err := redisClient.Set(key, value, expire_in).Err(); err != nil {
			error = utility.NewError("Set operation failed", utility.BackendStoreServiceOperationFailed)
			return
		} else {
			return
		}
	} else {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	}
}

func (b *BackendStoreService) Get(key string) (result string, error error) {

	result = ""
	error = nil

	if !b.IsServiceAvailable() {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	} else if DefaultBackendStore == Redis {
		if json, err := redisClient.Get(key).Result(); err != nil {
			error = utility.NewError("Set operation failed", utility.BackendStoreServiceOperationFailed)
			return
		} else {
			result = json
			return
		}
	} else {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	}
}

func (b *BackendStoreService) Del(key string) (error error) {

	error = nil

	if !b.IsServiceAvailable() {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	} else if DefaultBackendStore == Redis {
		if err := redisClient.Del(key).Err(); err != nil {
			error = utility.NewError("Set operation failed", utility.BackendStoreServiceOperationFailed)
			return
		} else {
			return
		}
	} else {
		error = utility.NewError("no backend are not available", utility.BackendStoreServiceErrorNotInitialized)
		return
	}
}
