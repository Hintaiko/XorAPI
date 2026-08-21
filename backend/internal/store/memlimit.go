package store

import (
	"sync"
	"time"
)

type counter struct {
	value int64
	until time.Time
}

var (
	mu       sync.Mutex
	counters = map[string]*counter{}
)

func memIncr(key string) int64 {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	c, ok := counters[key]
	if !ok || now.After(c.until) {
		c = &counter{until: now.Add(48 * time.Hour)}
		counters[key] = c
	}
	c.value++
	return c.value
}

func memAllow(key string, limit int64) bool {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	wkey := key + ":" + now.Format("200601021504")
	c, ok := counters[wkey]
	if !ok || now.After(c.until) {
		c = &counter{until: now.Add(time.Minute)}
		counters[wkey] = c
	}
	c.value++
	return c.value <= limit
}
