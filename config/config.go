package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

var errIncorrectLimit = errors.New("incorrect value of the limit")

type ServiceConfig struct {
	Port                     int               `json:"port"`
	OpenExchangeApiToken     string            `json:"openexchange_api_token"`
	OpenExchangeBaseUrl      string            `json:"openexchange_base_url"`
	TenorBaseUrl             string            `json:"tenor_base_url"`
	TenorApiToken            string            `json:"tenor_api_token"`
	TenorMediaStorageBaseUrl string            `json:"tenor_media_storage_base_url"`
	TenorSearchQueryLimit    int               `json:"tenor_search_query_limit"`
	RedisClientOptions       RedisClientConfig `json:"redis_client_options"`
	BaseCurrencyId           string            `json:"base_currency_id"`
}

type RedisClientConfig struct {
	DB       int    `json:"db"`
	Password string `json:"password"`
	Addr     string `json:"addr"`
}

var Config = new(ServiceConfig)

func init() {
	confContent, err := os.ReadFile("config/config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(confContent, Config); err != nil {
		log.Fatal(err)
	}
	if Config.TenorSearchQueryLimit < 0 {
		log.Fatal(errIncorrectLimit)
	}
}
