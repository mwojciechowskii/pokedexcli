package pokecache

import (
	"sync"
	"time"
)

type cacheEntry struct{
	createdAt time.Time
	val []byte
}

type Cache struct{
	entries map[string]cacheEntry
	mu sync.Mutex
}

func NewCache(interval time.Duration) *Cache{
	c := &Cache{
		entries: make(map[string]cacheEntry),	
	} 
	go c.reapLoop(interval)
	return c
}

func (c *Cache) Add (key string, val []byte){
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{
		createdAt: time.Now(),
		val: val,
	}
}

func (c *Cache) Get (key string) ([]byte, bool){
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.entries[key]
	if ok{
		return val.val, true
	}
	return nil, false
}


func (c *Cache) reapLoop(interval time.Duration){
	tick := time.NewTicker(interval)
	defer tick.Stop()
	
	//eeeeee
	for range tick.C{
		now := time.Now()
		c.mu.Lock()
		for key, entry := range c.entries{
			if now.Sub(entry.createdAt) > interval{
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
