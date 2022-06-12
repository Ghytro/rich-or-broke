package openexchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Ghytro/ab_interview/common"
	"github.com/Ghytro/ab_interview/config"
	"github.com/go-redis/redis"
)

var errIncorrectDate = errors.New("incorrect date")
var errIncorrectBaseCurrency = errors.New("incorrect base currency")

var redisClient = redis.NewClient(&redis.Options{
	DB:       config.Config.RedisClientOptions.DB,
	Password: config.Config.RedisClientOptions.Password,
	Addr:     config.Config.RedisClientOptions.Addr,
})

func init() {
	if redisClient == nil {
		log.Fatal("redis client is nil")
	}
	if err := redisClient.Ping().Err(); err != nil {
		log.Fatal(err)
	}
}

func getHistoricalRates(timestamp time.Time, base string) (map[string]float64, error) {
	date := timestamp.Format("2006-01-02")
	resp, err := http.Get(
		fmt.Sprintf(
			"%shistorical/%s.json?app_id=%s&base=%s",
			config.Config.OpenExchangeBaseUrl,
			date,
			config.Config.OpenExchangeApiToken,
			base,
		),
	)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusForbidden:
		return nil, errIncorrectBaseCurrency
	case http.StatusBadRequest:
		return nil, errIncorrectDate
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	unmarshaled := new(
		struct {
			Rates map[string]interface{} `json:"rates"`
		},
	)
	if err := json.Unmarshal(respBody, unmarshaled); err != nil {
		return nil, err
	}
	result := make(map[string]float64)
	for currency, rate := range unmarshaled.Rates {
		result[currency] = rate.(float64)
	}
	return result, nil
}

func HistoricalRates(timestamp time.Time, base string) (map[string]float64, error) {
	date := timestamp.Format("2006-01-02")
	redisCacheKey := fmt.Sprintf("openexchange_cache:%s:%s", date, base)
	cacheData, err := redisClient.HGetAll(redisCacheKey).Result()
	if err != nil {
		return nil, err
	}
	if len(cacheData) == 0 {
		// check if the correct currency was passed
		if !common.CurrencyExists(base) {
			return nil, errIncorrectBaseCurrency
		}
		rates, err := getHistoricalRates(timestamp, base)
		if err != nil {
			return nil, err
		}
		redisRates := make(map[string]interface{})
		for k, v := range rates {
			redisRates[k] = interface{}(v)
		}
		redisPipe := redisClient.Pipeline()
		redisPipe.HMSet(redisCacheKey, redisRates)
		redisPipe.Expire(redisCacheKey, 10*time.Minute)
		if _, err := redisPipe.Exec(); err != nil {
			return nil, err
		}
		return rates, nil
	}
	result := make(map[string]float64)
	for k, v := range cacheData {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		result[k] = f
	}
	return result, nil
}
