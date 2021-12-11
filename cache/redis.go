package cache

import (
	"crypto/tls"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func GetRedisClient() *redis.Client {
	var client = redis.NewClient(&redis.Options{
		Addr:      os.Getenv("REDIS_ADDR"),
		TLSConfig: getTLSConfig(),
		Password:  os.Getenv("REDIS_PASSWORD"),
		DB:        getDb(),
	})

	return client
}

func getTLSConfig() *tls.Config {
	tlsBool, err := strconv.ParseBool(os.Getenv("REDIS_TLS"))
	if err != nil {
		log.Panic("tlsBool convert string to bool error")
	}
	if tlsBool {
		tlsConfig := &tls.Config{}
		tlsSkipVerifyBool, err := strconv.ParseBool(os.Getenv("REDIS_TLS_SKIP_VERIFY"))
		if err != nil {
			log.Panic("tlsSkipVerifyBool convert string to bool error")
		}
		if tlsSkipVerifyBool {
			tlsConfig.InsecureSkipVerify = true
			return tlsConfig
		}
		tlsConfig.MinVersion = tls.VersionTLS12
		return tlsConfig
	}
	return nil
}

func getDb() int {
	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalf("cache redis getDb error:%v", err)
	}
	return db
}
