package common

import (
	"strings"
	"sync"
	"time"
)

type redisHealth struct {
	isAvailable              bool
	lastUnavailableTimestamp time.Time
	m                        sync.Mutex
}

var rh = &redisHealth{true, time.Time{}, sync.Mutex{}}

func IsBadRedisConnectionErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "dial tcp")
}

func SetRedisUnavailable() {
	rh.m.Lock()
	defer rh.m.Unlock()
	rh.isAvailable = false
	rh.lastUnavailableTimestamp = time.Now()
}

func IsRedisAvailable() bool {
	rh.m.Lock()
	defer rh.m.Unlock()
	if time.Since(rh.lastUnavailableTimestamp) > time.Minute {
		rh.isAvailable = true
	}
	return rh.isAvailable
}
