package ratelimiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupBenchmarkRedis creates a miniredis instance and client for benchmarking
func setupBenchmarkRedis(b *testing.B) (*redis.Client, *miniredis.Miniredis) {
	b.Helper()

	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

// BenchmarkFixedWindow_Allow benchmarks single request rate limiting
func BenchmarkFixedWindow_Allow(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     10000,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_AllowN benchmarks batch request rate limiting
func BenchmarkFixedWindow_AllowN(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     100000,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_AllowParallel benchmarks concurrent rate limiting
func BenchmarkFixedWindow_AllowParallel(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     1000000,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_Reset benchmarks reset operations
func BenchmarkFixedWindow_Reset(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     100,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_WindowSizes benchmarks different window sizes
func BenchmarkFixedWindow_WindowSizes(b *testing.B) {
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
			client, mr := setupBenchmarkRedis(b)
			defer mr.Close()

			config := &Config{
				Algorithm: FixedWindow,
				Limit:     10000,
				Window:    w.window,
			}

			limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_MultipleKeys benchmarks operations across different keys
func BenchmarkFixedWindow_MultipleKeys(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     1000,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_Denied benchmarks denied requests (over limit)
func BenchmarkFixedWindow_Denied(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     1,
		Window:    time.Hour, // Long window to keep it over limit
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_FailOpen benchmarks fail-open behavior
func BenchmarkFixedWindow_FailOpen(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     100,
		Window:    time.Minute,
		FailOpen:  true,
	}

	limiter, err := NewFixedWindow(client, config)
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

// BenchmarkFixedWindow_AllowWithResult benchmarks and validates result fields
func BenchmarkFixedWindow_AllowWithResult(b *testing.B) {
	client, mr := setupBenchmarkRedis(b)
	defer mr.Close()

	config := &Config{
		Algorithm: FixedWindow,
		Limit:     10000,
		Window:    time.Minute,
	}

	limiter, err := NewFixedWindow(client, config)
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
