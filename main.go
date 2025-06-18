package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/willf/bloom"
)

var ctx = context.Background()

// Simple cache struct with bloom filter and Redis client
type BloomCache struct {
	bf     *bloom.BloomFilter
	client *redis.Client
}

func NewBloomCache(redisAddr string, expectedEntries uint, falsePositiveRate float64) *BloomCache {
	bf := bloom.NewWithEstimates(expectedEntries, falsePositiveRate)
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	return &BloomCache{
		bf:     bf,
		client: rdb,
	}
}

func (bc *BloomCache) Set(key string, value string, expireSeconds int) error {
	// Put in Redis cache
	err := bc.client.Set(ctx, key, value, time.Duration(expireSeconds)*time.Second).Err()
	if err != nil {
		return err
	}
	// Add key to Bloom filter
	bc.bf.AddString(key)
	return nil
}

func (bc *BloomCache) Get(key string, bloomEnabled bool) (string, bool, error) {
	// First: Check bloom filter
	if bloomEnabled && !bc.bf.TestString(key) {
		// fmt.Println("[BLOOM-FILTER] not found ")
		// Definitely not in cache
		return "", false, nil
	}
	// fmt.Println("[BLOOM-FILTER] maybe found ")
	// Might be in cache, check Redis
	val, err := bc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key not found
		return "", false, nil
	}
	if err != nil {
		// Redis error
		return "", false, err
	}
	// Cache hit
	return val, true, nil
}

func main() {
	startTime := time.Now()

	// Example usage
	cache := NewBloomCache("localhost:6379", 1000000, 0.01) // 1 million entries, 1% fp rate

	TOTAL_ENTRIES := 100000
	BLOOM_FILTER_ENABLED := true

	entries := make([]struct {
		key   string
		value string
		ttl   int
	}, TOTAL_ENTRIES)

	for i := 0; i < TOTAL_ENTRIES; i++ {
		entries[i].key = fmt.Sprintf("key%d", i)
		entries[i].value = fmt.Sprintf("value%d", i)
		entries[i].ttl = 60 + i // Example: different TTLs
	}

	for _, entry := range entries {
		if err := cache.Set(entry.key, entry.value, entry.ttl); err != nil {
			panic(err)
		}
	}

	// Randomly try to get elements in the entries list
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < TOTAL_ENTRIES/2; i++ {
		// Randomly pick an index, with a chance to miss (i.e., pick an out-of-range index)
		var idx int
		if rand.Float64() < 0.8 { // 80% chance to pick a valid index
			idx = rand.Intn(len(entries))
		} else { // 20% chance to pick a missing (non-existent) index
			idx = len(entries) + rand.Intn(1000) // index outside the valid range
		}
		key := fmt.Sprintf("key%d", idx)
		_, found, err := cache.Get(key, BLOOM_FILTER_ENABLED)
		if err != nil {
			panic(err)
		}
		if found {
			// fmt.Printf("Found in cache: key=%s, value=%s\n", key, value)
		} else {
			// fmt.Printf("Not found in cache: key=%s\n", key)
		}
	}

	elapsed := time.Since(startTime)
	if BLOOM_FILTER_ENABLED {
		fmt.Println("Bloom filter enabled")
	} else {
		fmt.Println("Bloom filter disabled")
	}

	fmt.Printf("Time taken: %s\n", elapsed)

}
