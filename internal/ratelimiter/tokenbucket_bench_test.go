package ratelimiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupBenchmarkRedisTokenBucket creates a miniredis instance and client for benchmarking
func setupBenchmarkRedisTokenBucket(b *testing.B) (*redis.Client, *miniredis.Miniredis) {
	b.Helper()

	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

// BenchmarkTokenBucket_Allow benchmarks single request rate limiting
func BenchmarkTokenBucket_Allow(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := limiter.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenBucket_AllowN benchmarks batch request rate limiting
func BenchmarkTokenBucket_AllowN(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:456"

	benchmarks := []struct {
		name string
		n    int64
	}{
		{"N=1", 1},
		{"N=10", 10},
		{"N=100", 100},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := limiter.AllowN(ctx, key, bm.n)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTokenBucket_AllowParallel benchmarks concurrent rate limiting
func BenchmarkTokenBucket_AllowParallel(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     1000000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench:user:%d", i%100) // 100 different keys
			_, err := limiter.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// BenchmarkTokenBucket_Reset benchmarks reset operations
func BenchmarkTokenBucket_Reset(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:reset"

	// Pre-populate with some requests
	for i := 0; i < 50; i++ {
		limiter.Allow(ctx, key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := limiter.Reset(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenBucket_WindowSizes benchmarks different window sizes
func BenchmarkTokenBucket_WindowSizes(b *testing.B) {
	windows := []struct {
		name   string
		window time.Duration
	}{
		{"1s", time.Second},
		{"1m", time.Minute},
		{"1h", time.Hour},
	}

	for _, w := range windows {
		b.Run(w.name, func(b *testing.B) {
			client, mr := setupBenchmarkRedisTokenBucket(b)
			defer mr.Close()

			config := &Config{
				Algorithm: TokenBucket,
				Limit:     10000,
				Window:    w.window,
			}

			limiter, err := NewTokenBucket(client, config)
			if err != nil {
				b.Fatal(err)
			}
			defer limiter.Close()

			ctx := context.Background()
			key := "bench:user:window"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := limiter.Allow(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTokenBucket_MultipleKeys benchmarks operations across different keys
func BenchmarkTokenBucket_MultipleKeys(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     1000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()

	keysets := []struct {
		name  string
		count int
	}{
		{"10keys", 10},
		{"100keys", 100},
		{"1000keys", 1000},
	}

	for _, ks := range keysets {
		b.Run(ks.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("bench:user:%d", i%ks.count)
				_, err := limiter.Allow(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTokenBucket_Denied benchmarks denied requests (over limit)
func BenchmarkTokenBucket_Denied(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     1,
		Window:    time.Hour, // Long window for slow refill
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:denied"

	// Use up the limit
	limiter.Allow(ctx, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
		if result.Allowed {
			b.Fatal("expected request to be denied")
		}
	}
}

// BenchmarkTokenBucket_FailOpen benchmarks fail-open behavior
func BenchmarkTokenBucket_FailOpen(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100,
		Window:    time.Minute,
		FailOpen:  true,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:failopen"

	// Close miniredis to simulate failure
	mr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := limiter.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenBucket_AllowWithResult benchmarks and validates result fields
func BenchmarkTokenBucket_AllowWithResult(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:result"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
		// Access result fields to ensure they're computed
		_ = result.Allowed
		_ = result.Remaining
		_ = result.ResetAt
	}
}

// BenchmarkTokenBucket_HighThroughput benchmarks high-throughput scenarios
func BenchmarkTokenBucket_HighThroughput(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100000, // High limit for throughput testing
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:throughput"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := limiter.AllowN(ctx, key, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenBucket_BurstAllowance benchmarks burst capacity handling
func BenchmarkTokenBucket_BurstAllowance(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     1000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each iteration uses a fresh key (new bucket starts full)
		key := fmt.Sprintf("bench:user:burst:%d", i)
		// Consume entire bucket at once (burst)
		_, err := limiter.AllowN(ctx, key, 1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenBucket_MixedLoad benchmarks mixed single and batch operations
func BenchmarkTokenBucket_MixedLoad(b *testing.B) {
	client, mr := setupBenchmarkRedisTokenBucket(b)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100000,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	if err != nil {
		b.Fatal(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "bench:user:mixed"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between single and batch requests
		if i%2 == 0 {
			_, err := limiter.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
		} else {
			_, err := limiter.AllowN(ctx, key, 5)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
