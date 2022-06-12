package tenor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/Ghytro/ab_interview/config"
	"github.com/go-redis/redis"
)

var errIncorrectLimit = errors.New("incorrect value of the limit")
var fallbackCounter = 0

var redisClient = redis.NewClient(&redis.Options{
	DB:       config.Config.RedisClientOptions.DB,
	Password: config.Config.RedisClientOptions.Password,
	Addr:     config.Config.RedisClientOptions.Addr,
})

type Gif struct {
	BinaryContent []byte
}

func GetRandomGif(searchQuery string, limit int) (*Gif, error) {
	if limit < 0 {
		return nil, errIncorrectLimit
	}

	resp, err := http.Get(
		fmt.Sprintf(
			"%ssearch?q=%s&key=%s&limit=%d",
			config.Config.TenorBaseUrl,
			searchQuery,
			config.Config.TenorApiToken,
			limit,
		),
	)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	unmarshaled := new(
		struct {
			Results []struct {
				Media []struct {
					Gif struct {
						Url string `json:"url"`
					} `json:"gif"`
				} `json:"media"`
			} `json:"results"`
		},
	)
	if err := json.Unmarshal(respBody, unmarshaled); err != nil {
		return nil, err
	}
	gifUrl := unmarshaled.Results[rand.Intn(len(unmarshaled.Results))].Media[0].Gif.Url
	splittedGifUrl := strings.Split(gifUrl, "/")
	gifId := splittedGifUrl[len(splittedGifUrl)-2]
	redisCacheKey := fmt.Sprintf("tenor_cache:gif:%s", gifId)
	gifBytes, err := redisClient.Get(redisCacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			fmt.Println(fallbackCounter)
			fallbackCounter++
			resp, err = http.Get(unmarshaled.Results[rand.Intn(len(unmarshaled.Results))].Media[0].Gif.Url)
			if err != nil {
				return nil, err
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body.Close()
			result := &Gif{respBody}
			redisClient.Set(redisCacheKey, respBody, 0)
			return result, nil
		}
	}
	return &Gif{gifBytes}, nil
}
