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

type entry struct {
	key   string
	value string
	ttl   int
}

func saveEntriesToRedis(cache *BloomCache) []entry {

	entries := make([]entry, TOTAL_ENTRIES)
	for i := 0; i < TOTAL_ENTRIES; i++ {
		entries[i] = entry{
			key:   fmt.Sprintf("key%d", i),
			value: strings.Repeat("X", 1024), // 1KB value
			ttl:   100000,
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
	return entries
}

const (
	TOTAL_ENTRIES = 400000
	GET_REQUESTS  = 200000
	NUM_RUNS      = 10
	REDIS_ADDR    = "localhost:6379"
)

func main() {

	cache := NewBloomCache(REDIS_ADDR, 1000000, 0.01) // 1 million entries, 1% fp rate

	// Use a single struct for entry to avoid repeated struct literal allocation
	entries := saveEntriesToRedis(cache)
	fmt.Println("Entries saved to Redis")

	// Run the benchmark n times for both Bloom filter enabled and disabled, report average times and speedup

	type benchResult struct {
		totalTime   time.Duration
		totalHits   int
		totalMisses int
	}

	runBenchmark := func(bloomEnabled bool) benchResult {
		var totalTime time.Duration
		var totalHits, totalMisses int

		for run := 0; run < NUM_RUNS; run++ {
			start := time.Now()
			cacheHits := 0
			cacheMisses := 0
			for i := 0; i < GET_REQUESTS; i++ {
				var idx int
				if rand.Float64() < 0.8 {
					idx = rand.Intn(len(entries))
				} else {
					idx = len(entries) + rand.Intn(1000)
				}
				key := fmt.Sprintf("key%d", idx)
				_, found, err := cache.Get(key, bloomEnabled)
				if err != nil {
					panic(err)
				}
				if found {
					cacheHits++
				} else {
					cacheMisses++
				}
			}

			elapsed := time.Since(start)

			totalTime += elapsed
			totalHits += cacheHits
			totalMisses += cacheMisses

			fmt.Printf("  Run %d: time=%s, hits=%d, misses=%d, hit rate=%.2f%%\n", run+1, elapsed, cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
		}
		return benchResult{
			totalTime:   totalTime,
			totalHits:   totalHits,
			totalMisses: totalMisses,
		}
	}

	fmt.Println("Running benchmark with Bloom filter ENABLED...")
	bloomOnResult := runBenchmark(true)
	fmt.Println("Running benchmark with Bloom filter DISABLED...")
	bloomOffResult := runBenchmark(false)

	avgTimeOn := bloomOnResult.totalTime / time.Duration(NUM_RUNS)
	avgTimeOff := bloomOffResult.totalTime / time.Duration(NUM_RUNS)

	fmt.Println("-------------------------")
	fmt.Printf("Bloom filter ENABLED:\n")
	fmt.Printf("  Avg time: %s\n", avgTimeOn)
	fmt.Printf("  Avg hit rate: %.2f%%\n", float64(bloomOnResult.totalHits)/float64(bloomOnResult.totalHits+bloomOnResult.totalMisses)*100)
	fmt.Println("-------------------------")
	fmt.Printf("Bloom filter DISABLED:\n")
	fmt.Printf("  Avg time: %s\n", avgTimeOff)
	fmt.Printf("  Avg hit rate: %.2f%%\n", float64(bloomOffResult.totalHits)/float64(bloomOffResult.totalHits+bloomOffResult.totalMisses)*100)
	fmt.Println("-------------------------")

	speedup := float64(avgTimeOff) / float64(avgTimeOn)
	fmt.Printf("Speedup with Bloom filter ON: %.2fx\n", speedup)

}
