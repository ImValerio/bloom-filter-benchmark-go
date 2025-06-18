package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
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

	cache := NewBloomCache("localhost:6379", 1000000, 0.01) // 1 million entries, 1% fp rate

	TOTAL_ENTRIES := 200000
	BLOOM_FILTER_ENABLED := true

	// Use a single struct for entry to avoid repeated struct literal allocation
	type entry struct {
		key   string
		value string
		ttl   int
	}

	entries := make([]entry, TOTAL_ENTRIES)
	for i := 0; i < TOTAL_ENTRIES; i++ {
		entries[i] = entry{
			key:   fmt.Sprintf("key%d", i),
			value: strings.Repeat("X", 1024), // 1KB value
			ttl:   120,
		}
	}

	// Batch set using goroutines for parallelism (limit concurrency to avoid overwhelming Redis)
	const maxConcurrency = 32
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var setErr error
	var once sync.Once

	for _, entry := range entries {
		sem <- struct{}{}
		wg.Add(1)
		go func(e struct {
			key   string
			value string
			ttl   int
		}) {
			defer wg.Done()
			if err := cache.Set(e.key, e.value, e.ttl); err != nil {
				once.Do(func() { setErr = err })
			}
			<-sem
		}(entry)
	}
	wg.Wait()
	if setErr != nil {
		panic(setErr)
	}

	// Randomly try to get elements in the entries list
	rand.New(rand.NewSource(time.Now().UnixNano()))
	cacheHits := 0
	cacheMisses := 0
	for i := 0; i < TOTAL_ENTRIES/2; i++ {
		// Randomly pick an index, with a chance to miss (i.e., pick an out-of-range index)
		var idx int
		if rand.Float64() < 0.8 { // 80% chance to pick a valid index
			idx = rand.Intn(len(entries))
		} else { // 20% chance to pick a missing (non-existent) index
			idx = len(entries) + rand.Intn(1000)
		}
		key := fmt.Sprintf("key%d", idx)
		_, found, err := cache.Get(key, BLOOM_FILTER_ENABLED)
		if err != nil {
			panic(err)
		}
		if found {
			cacheHits++
		} else {
			cacheMisses++
		}
	}

	elapsed := time.Since(startTime)

	fmt.Printf("Cache hits: %d\n", cacheHits)
	fmt.Printf("Cache misses: %d\n", cacheMisses)
	fmt.Printf("Cache hit rate: %.2f%%\n", float64(cacheHits)/float64(cacheHits+cacheMisses)*100)

	fmt.Println("-------------------------")

	if BLOOM_FILTER_ENABLED {
		fmt.Println("Bloom filter enabled")
	} else {
		fmt.Println("Bloom filter disabled")
	}

	fmt.Printf("Time taken: %s\n", elapsed)

}
