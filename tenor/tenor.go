package tenor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Ghytro/ab_interview/common"
	"github.com/Ghytro/ab_interview/config"
	"github.com/go-redis/redis"
)

var errNoGifIdsInCache = errors.New("no gif ids found in cache")
var errNoGifInCache = errors.New("no gif with the given id in cache")
var ErrIncorrectTenorToken = errors.New("incorrect token provided to tenor api")

var redisClient = redis.NewClient(&redis.Options{
	DB:          config.Config.RedisClientOptions.DB,
	Password:    config.Config.RedisClientOptions.Password,
	Addr:        config.Config.RedisClientOptions.Addr,
	ReadTimeout: time.Millisecond * 100,
})

type Gif struct {
	BinaryContent []byte
}

func getRandomGifIdFromCache(searchQuery string) (string, error) {
	redisCacheKey := fmt.Sprintf("tenor_cache:gif_ids:%s", searchQuery)
	gifId, err := redisClient.SRandMember(redisCacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errNoGifIdsInCache
		}
		return "", err
	}
	return gifId, nil
}

func addGifIdsToCache(searchQuery string, gifIds ...string) {
	redisCacheKey := fmt.Sprintf("tenor_cache:gif_ids:%s", searchQuery)
	pipe := redisClient.Pipeline()
	for _, id := range gifIds {
		pipe.SAdd(redisCacheKey, id)
	}
	pipe.Expire(redisCacheKey, time.Hour*24)
	pipe.Exec()
}

func getSearchQueryGifIdsFromApi(searchQuery string) ([]string, error) {
	resp, err := http.Get(
		fmt.Sprintf(
			"%ssearch?q=%s&key=%s&limit=%d",
			config.Config.TenorBaseUrl,
			searchQuery,
			config.Config.TenorApiToken,
			config.Config.TenorSearchQueryLimit,
		),
	)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrIncorrectTenorToken
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
	result := make([]string, len(unmarshaled.Results))
	for i, r := range unmarshaled.Results {
		gifUrl := r.Media[0].Gif.Url
		splittedGifUrl := strings.Split(gifUrl, "/")
		gifId := splittedGifUrl[len(splittedGifUrl)-2]
		result[i] = gifId
	}
	return result, nil
}

func getRandomGifId(searchQuery string) (string, error) {
	if !common.IsRedisAvailable() {
		gifIds, err := getSearchQueryGifIdsFromApi(searchQuery)
		common.LogIfVerbose("tenor.getRandomGifId: redis not available, falling back to api")
		return gifIds[rand.Intn(len(gifIds))], err
	}
	gifId, err := getRandomGifIdFromCache(searchQuery)
	if err != nil {
		switch {
		case common.IsBadRedisConnectionErr(err):
			common.SetRedisUnavailable()
			gifIds, err := getSearchQueryGifIdsFromApi(searchQuery)
			common.LogIfVerbose("tenor.getRandomGifId: bad connection with redis, setting unavailable")
			return gifIds[rand.Intn(len(gifIds))], err
		case err == errNoGifIdsInCache:
			gifIds, err := getSearchQueryGifIdsFromApi(searchQuery)
			if err != nil {
				return "", err
			}
			addGifIdsToCache(searchQuery, gifIds...)
			common.LogIfVerbose("tenor.getRandomGifId: no gif ids in cache for the query, adding")
			return gifIds[rand.Intn(len(gifIds))], nil
		default:
			return "", err
		}
	}
	common.LogIfVerbose("tenor.getRandomGifId: returning gif id from cache")
	return gifId, nil
}

func getGifByIdFromCache(gifId string) (*Gif, error) {
	redisCacheKey := fmt.Sprintf("tenor_cache:gif:%s", gifId)
	gifBytes, err := redisClient.Get(redisCacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errNoGifInCache
		}
		return nil, err
	}
	return &Gif{gifBytes}, nil
}

func addGifToCache(gifId string, gif *Gif) {
	redisClient.Set(
		fmt.Sprintf("tenor_cache:gif:%s", gifId),
		gif.BinaryContent,
		0,
	)
}

func getGifByIdFromTenorApi(gifId string) (*Gif, error) {
	resp, err := http.Get(
		fmt.Sprintf(
			"%s%s/tenor.gif",
			config.Config.TenorMediaStorageBaseUrl,
			gifId,
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
	return &Gif{respBody}, nil
}

func getGifById(gifId string) (*Gif, error) {
	if !common.IsRedisAvailable() {
		common.LogIfVerbose("tenor.getGifById: redis not available, falling back to api")
		return getGifByIdFromTenorApi(gifId)
	}
	gif, err := getGifByIdFromCache(gifId)
	if err != nil {
		switch {
		case common.IsBadRedisConnectionErr(err):
			common.SetRedisUnavailable()
			common.LogIfVerbose("tenor.getGifById: bad connection with redis, setting unavailable")
			return getGifByIdFromTenorApi(gifId)
		case err == errNoGifInCache:
			gif, err = getGifByIdFromTenorApi(gifId)
			if err != nil {
				return nil, err
			}
			addGifToCache(gifId, gif)
			common.LogIfVerbose("tenor.getGifById: no gif found in cache with this id, adding")
			return gif, nil
		default:
			return nil, err
		}
	}
	common.LogIfVerbose("tenor.getGifById: returning gif from cache")
	return gif, nil
}

func GetRandomGif(searchQuery string) (*Gif, error) {
	searchQuery = strings.ReplaceAll(searchQuery, " ", "+")
	gifId, err := getRandomGifId(searchQuery)
	if err != nil {
		return nil, err
	}
	return getGifById(gifId)
}
