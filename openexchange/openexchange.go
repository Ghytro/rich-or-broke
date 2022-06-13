package openexchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Ghytro/ab_interview/config"
	"github.com/go-redis/redis"
)

var ErrIncorrectDate = errors.New("incorrect date")
var ErrIncorrectBaseCurrency = errors.New("incorrect base currency")
var ErrNoRatesDataInCache = errors.New("no rates data in cache by given date and base")
var ErrIncorrectOpenExchangeToken = errors.New("incorrect access token provided to openexchange")

var redisClient = redis.NewClient(&redis.Options{
	DB:       config.Config.RedisClientOptions.DB,
	Password: config.Config.RedisClientOptions.Password,
	Addr:     config.Config.RedisClientOptions.Addr,
})

func getHistoricalRatesFromCache(date string, base string) (map[string]float64, error) {
	redisCacheKey := fmt.Sprintf("openexchange_cache:%s:%s", date, base)
	cacheData, err := redisClient.HGetAll(redisCacheKey).Result()
	if err != nil {
		return nil, err
	}
	if len(cacheData) == 0 {
		return nil, ErrNoRatesDataInCache
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

func addRateToCache(date, base string, rates map[string]float64) error {
	redisRates := make(map[string]interface{})
	for k, v := range rates {
		redisRates[k] = interface{}(v)
	}
	redisPipe := redisClient.Pipeline()
	redisCacheKey := fmt.Sprintf("openexchange_cache:%s:%s", date, base)
	redisPipe.HMSet(redisCacheKey, redisRates)
	redisPipe.Expire(redisCacheKey, 10*time.Minute)
	if _, err := redisPipe.Exec(); err != nil {
		return err
	}
	return nil
}

func getHistoricalRatesFromApi(date string, base string) (map[string]float64, error) {
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
		return nil, ErrIncorrectBaseCurrency
	case http.StatusBadRequest:
		return nil, ErrIncorrectDate
	case http.StatusUnauthorized:
		return nil, ErrIncorrectOpenExchangeToken
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
	rates, err := getHistoricalRatesFromCache(date, base)
	if err != nil {
		if err == ErrNoRatesDataInCache {
			rates, err = getHistoricalRatesFromApi(date, base)
			if err != nil {
				return nil, err
			}
			addRateToCache(date, base, rates)
			return rates, nil
		}
		return nil, err
	}
	return rates, nil
}
