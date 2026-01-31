# Rate Limiting Algorithms

This document provides an in-depth analysis of the three rate limiting algorithms implemented in this project.

## Quick Comparison

| Aspect | Fixed Window | Sliding Window | Token Bucket |
|--------|--------------|----------------|--------------|
| **Complexity** | Low | Medium | High |
| **Accuracy** | Low (boundary issues) | High (most precise) | Medium (smooth) |
| **Memory Usage** | Very Low (1 counter) | Medium (2 counters) | Low (2 fields) |
| **Redis Operations** | 1 key per window | 2 keys (current + previous) | 1 key with hash |
| **Burst Handling** | Poor (allows 2x at boundary) | Good (mathematically smooth) | Excellent (explicit burst capacity) |
| **Implementation** | Simplest | Moderate | Complex (refill logic) |
| **Edge Cases** | Boundary gaming | Clock skew sensitivity | Float precision issues |
| **Best For** | Internal quotas, simple limits | Billing, strict SLAs | APIs, variable traffic |
| **Performance (RPS)** | ~50K | ~30K | ~35K |

## Algorithm Selection Guide

```
START HERE: What's your main concern?

┌─────────────────────────────────┐
│  Need absolute simplicity?      │──YES──> Fixed Window
│  (internal services, soft caps) │
└─────────────────────────────────┘
         │ NO
         ▼
┌─────────────────────────────────┐
│  Money/SLA involved?            │──YES──> Sliding Window
│  (billing, strict enforcement)  │
└─────────────────────────────────┘
         │ NO
         ▼
┌─────────────────────────────────┐
│  Variable traffic patterns?     │──YES──> Token Bucket
│  (APIs, user-facing features)   │
└─────────────────────────────────┘
         │ NO
         ▼
    Default: Token Bucket
    (industry standard, good all-around)
```

---

## Fixed Window Counter

### Concept

The simplest rate limiting algorithm. Maintains a counter for each time window, resets when window expires.

### Visual Representation

```
Window 1: 10:00-10:01        Window 2: 10:01-10:02
Limit: 5 requests            Limit: 5 requests

10:00:00  ✓ (1/5)  ║
10:00:15  ✓ (2/5)  ║
10:00:30  ✓ (3/5)  ║
10:00:45  ✓ (4/5)  ║
10:00:59  ✓ (5/5)  ║  ← Window edge
─────────────────────║─────────────────  (Counter resets)
10:01:00  ✓ (1/5)  ║
10:01:01  ✓ (2/5)  ║
10:01:02  ✓ (3/5)  ║

Problem: User can make 5 requests at 10:00:59 and 5 more at 10:01:00
         = 10 requests in 1 second! (boundary gaming)
```

### How It Works

**Algorithm:**
```
1. Calculate window_key = floor(current_time / window_duration) * window_duration
   Example: floor(10:00:35 / 60) * 60 = 10:00:00

2. counter = INCR "ratelimit:user:123:10:00:00"

3. If counter == 1 (first request in window):
     SET TTL window_duration + buffer

4. If counter <= limit:
     ALLOW (remaining = limit - counter)
   Else:
     DENY (retry_after = TTL of key)
```

**Lua Script:**
```lua
-- fixed_window.lua
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call('INCR', key)

if current == 1 then
    redis.call('EXPIRE', key, window)
end

local ttl = redis.call('TTL', key)

if current > limit then
    return {0, limit, 0, ttl}  -- denied
else
    return {1, limit, limit - current, ttl}  -- allowed
end
```

### Redis Data Structure

```
Key:   "ratelimit:user:123:1704448800"  (timestamp rounded to window)
Type:  String (integer counter)
Value: 5
TTL:   60 seconds (window duration + buffer)

Example data:
"ratelimit:user:alice:1704448800" → "3"  (TTL: 45s)
"ratelimit:user:bob:1704448860"   → "7"  (TTL: 55s)
```

### The Boundary Problem

**Scenario:** Limit is 100 requests/minute

```
10:00:59 → User sends 100 requests → All ALLOWED (counter = 100)
10:01:00 → Window resets (counter = 0)
10:01:00 → User sends 100 requests → All ALLOWED (counter = 100)

Result: 200 requests in 1 second! (2x the limit)
```

**Why this happens:**
- Windows are aligned to wall clock (10:00:00, 10:01:00, etc.)
- No overlap between windows
- Hard reset at boundary allows fresh quota

**Mitigation:**
- Use shorter windows (5-second windows are harder to game)
- Accept the limitation for non-critical use cases
- Use Sliding Window for strict enforcement

### Pros & Cons

**✅ Pros:**
- **Simplest implementation** - Single INCR + EXPIRE
- **Fastest** - ~50K RPS per Redis instance
- **Lowest memory** - ~100 bytes per user
- **Easy to reason about** - "X requests per window, period"
- **Easy to debug** - Just look at counter value
- **Predictable reset** - Always at wall clock boundaries

**❌ Cons:**
- **Boundary gaming** - Can get 2x limit at window edge
- **Unfair distribution** - Burst at start of window blocks entire window
- **Clock dependency** - Servers must have synchronized clocks (NTP)
- **Not suitable for strict SLAs** - Boundary problem violates guarantees

### Use Cases

1. **Internal Service Limits**
   ```go
   // Limit internal API calls between microservices
   config := &Config{
       Algorithm: FixedWindow,
       Limit:     10000,
       Window:    time.Minute,
   }
   ```
   - Gaming not a concern (trusted services)
   - Simplicity valued over precision

2. **Soft Quotas**
   ```go
   // Daily email sending limit
   config := &Config{
       Algorithm: FixedWindow,
       Limit:     1000,
       Window:    24 * time.Hour,
   }
   ```
   - Boundary gaming minimal impact over 24-hour window

3. **High-Throughput Systems**
   - When every millisecond counts
   - Minimal overhead critical

4. **Development/Testing**
   - Easy to understand and debug
   - Quick to implement

### Performance Characteristics

**Throughput:** ~50,000 RPS (single Redis instance)
**Latency:** 0.5-1ms (Redis script execution)
**Memory:** ~100 bytes per active user

---

## Sliding Window Counter

### Concept

Maintains two windows (current and previous) and calculates weighted average based on current position within the window. Prevents boundary gaming.

### Visual Representation

```
Current time: 10:00:30 (30 seconds into current window)
Window: 1 minute (60 seconds)

Previous Window          Current Window
09:59:30 - 10:00:30     10:00:30 - 10:01:30
Had 8 requests          Has 3 requests so far
        ▼                       ▼
    [████████]              [███░░░░░░]
                     ▲
                We're here (10:00:30)
                30 seconds into current window

Calculation:
  Weight of previous window: (60-30)/60 = 0.5
  Weight of current window: 30/60 = 0.5

  Effective count = (8 * 0.5) + (3 * 1.0) = 4 + 3 = 7

  If limit is 10: ALLOW (7 < 10)
  If limit is 6:  DENY (7 > 6)
```

### How It Works

**Algorithm:**
```
1. current_window_key = floor(now / window) * window
   previous_window_key = current_window_key - window

2. current_count = GET current_window_key (or 0)
   previous_count = GET previous_window_key (or 0)

3. elapsed_in_current = now - current_window_key
   overlap_weight = (window - elapsed_in_current) / window

4. weighted_count = (previous_count * overlap_weight) + current_count

5. If weighted_count < limit:
     INCR current_window_key
     ALLOW
   Else:
     DENY
```

**Lua Script:**
```lua
-- sliding_window.lua
local current_key = KEYS[1]
local previous_key = KEYS[2]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Get counts
local current_count = tonumber(redis.call('GET', current_key) or 0)
local previous_count = tonumber(redis.call('GET', previous_key) or 0)

-- Calculate window start
local current_window_start = math.floor(now / window) * window
local elapsed = now - current_window_start
local overlap_weight = (window - elapsed) / window

-- Weighted count
local weighted_count = (previous_count * overlap_weight) + current_count

if weighted_count < limit then
    -- Allow: increment current window
    current_count = redis.call('INCR', current_key)

    if current_count == 1 then
        redis.call('EXPIRE', current_key, window * 2)
    end

    local remaining = limit - math.ceil(weighted_count) - 1
    return {1, limit, remaining, window - elapsed}  -- allowed
else
    -- Deny
    return {0, limit, 0, window - elapsed}  -- denied
end
```

### Redis Data Structure

```
Current Window:
Key:   "ratelimit:user:123:current:1704448860"
Type:  String (integer counter)
Value: 25
TTL:   120 seconds (2x window duration)

Previous Window:
Key:   "ratelimit:user:123:previous:1704448800"
Type:  String (integer counter)
Value: 80
TTL:   60 seconds (window duration)

Example:
"ratelimit:user:alice:current:1704448860"  → "25" (TTL: 100s)
"ratelimit:user:alice:previous:1704448800" → "80" (TTL: 45s)
```

### Mathematical Precision

**Scenario:** Limit is 100 requests/minute, attempting boundary gaming

```
10:00:59 → User sends 100 requests
    Current window [10:00:00-10:01:00]: 100 requests
    Previous window [09:59:00-10:00:00]: 0 requests
    → All 100 ALLOWED

10:01:00 → User tries to send 100 more
    Current window [10:01:00-10:02:00]: 0 requests
    Previous window [10:00:00-10:01:00]: 100 requests

    Elapsed in current: 0 seconds
    Overlap weight: (60 - 0) / 60 = 1.0

    Weighted count = (100 * 1.0) + 0 = 100

    100 >= 100 limit → DENY (all 100 requests)

10:01:30 → User tries again
    Elapsed: 30 seconds
    Overlap weight: (60 - 30) / 60 = 0.5

    Weighted count = (100 * 0.5) + 0 = 50

    50 < 100 limit → Can allow 50 more requests
```

**Result:** Boundary gaming prevented! Smooth transition between windows.

### Pros & Cons

**✅ Pros:**
- **Prevents boundary gaming** - Weighted average smooths window transition
- **More accurate** - True requests-per-window enforcement
- **Still efficient** - Only 2 Redis keys
- **Precise** - Mathematical guarantee of limit
- **Fair** - No advantage to timing requests at window edges

**❌ Cons:**
- **More complex** - Weighted calculation required
- **Slower than Fixed Window** - ~30K RPS vs 50K RPS
- **2x memory** - Two keys per user
- **Clock skew sensitive** - Requires accurate timestamps
- **Not perfect for bursts** - Smooth distribution might not match usage patterns

### Use Cases

1. **Billing/Monetization**
   ```go
   // API calls with overage charges
   config := &Config{
       Algorithm: SlidingWindow,
       Limit:     10000,
       Window:    time.Hour,
   }
   ```
   - Accuracy critical (money involved)
   - Can't allow boundary gaming (disputes/abuse)

2. **SLA Enforcement**
   ```go
   // Contractual rate limits
   config := &Config{
       Algorithm: SlidingWindow,
       Limit:     100,
       Window:    time.Second,
   }
   ```
   - Legal/contractual requirements
   - Must strictly enforce limits

3. **Abuse Prevention**
   - Prevents gaming of limits
   - Security-critical rate limits

4. **Public APIs (Premium)**
   - Professional appearance
   - Predictable, fair behavior

### Performance Characteristics

**Throughput:** ~30,000 RPS (single Redis instance)
**Latency:** 0.8-1.5ms (Redis script + calculation)
**Memory:** ~200 bytes per active user (2 keys)

---

## Token Bucket

### Concept

Imagine a bucket that holds tokens. Tokens refill at a constant rate. Each request consumes one token. Allows bursts up to bucket capacity while maintaining average rate.

### Visual Representation

```
Bucket capacity: 10 tokens
Refill rate: 1 token per second (10 tokens per 10 seconds)

Timeline:
10:00:00 - Bucket starts full [██████████] 10 tokens

10:00:01 - Request uses 1 token [█████████░] 9 tokens
10:00:02 - +1 refill, request uses 1 [█████████░] 9 tokens
10:00:03 - +1 refill, request uses 1 [█████████░] 9 tokens

...

10:00:10 - All tokens consumed [░░░░░░░░░░] 0 tokens → DENY
10:00:11 - +1 refill [█░░░░░░░░░] 1 token → ALLOW (if request comes)
10:00:12 - +1 refill [██░░░░░░░░] 2 tokens

Burst scenario:
10:00:00 - User quiet for 10 seconds
         - Bucket refills to capacity [██████████] 10 tokens
10:00:00 - 10 rapid requests arrive
         - All 10 ALLOWED instantly [░░░░░░░░░░] 0 tokens
10:00:10 - 10 seconds pass, 10 refills [██████████] 10 tokens
10:00:10 - Another burst of 10 [░░░░░░░░░░] 0 tokens (all allowed)

Average rate: 10 tokens / 10 seconds = 1 token/sec (sustained)
Burst capacity: 10 tokens (allows occasional spikes)
```

### How It Works

**Algorithm:**
```
1. Load bucket state from Redis:
     tokens = current_tokens
     last_refill = last_refill_timestamp

2. Calculate refill:
     elapsed = now - last_refill
     refill_rate = capacity / window
     tokens_to_add = elapsed * refill_rate
     tokens = min(tokens + tokens_to_add, capacity)

3. Check if request allowed:
     If tokens >= 1:
       tokens -= 1
       ALLOW (remaining = floor(tokens))
     Else:
       DENY (retry_after = time until next token)

4. Save state:
     HSET key "tokens" tokens "last_refill" now
```

**Lua Script:**
```lua
-- token_bucket.lua
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4] or 1)

-- Get current state
local bucket = redis.call('HGETALL', key)
local tokens = capacity
local last_refill = now

if #bucket > 0 then
    for i = 1, #bucket, 2 do
        if bucket[i] == 'tokens' then
            tokens = tonumber(bucket[i + 1])
        elseif bucket[i] == 'last_refill' then
            last_refill = tonumber(bucket[i + 1])
        end
    end
end

-- Calculate refill
local elapsed = math.max(0, now - last_refill)
local refill_rate = capacity / window
local tokens_to_add = elapsed * refill_rate
tokens = math.min(tokens + tokens_to_add, capacity)

-- Check if we can allow the request
if tokens >= requested then
    tokens = tokens - requested

    -- Update state
    redis.call('HSET', key, 'tokens', tokens, 'last_refill', now)
    redis.call('EXPIRE', key, window * 2)

    local remaining = math.floor(tokens)
    return {1, capacity, remaining, 0}  -- allowed
else
    -- Calculate retry_after
    local needed = requested - tokens
    local retry_after = math.ceil(needed / refill_rate)

    return {0, capacity, 0, retry_after}  -- denied
end
```

### Redis Data Structure

```
Key:  "ratelimit:user:123"
Type: Hash
Fields:
  tokens: 7.5 (float - current token count)
  last_refill: 1704448825 (timestamp of last refill calculation)
TTL:  120 seconds (2x window duration)

Example data:
HGETALL "ratelimit:user:alice"
1) "tokens"
2) "7.5"
3) "last_refill"
4) "1704448825"
```

### Burst Behavior Deep Dive

**Configuration:**
- Capacity: 100 tokens
- Window: 60 seconds
- Refill rate: 100/60 = 1.67 tokens/second

**Scenario 1: Normal usage**
```
User makes 1 request per second (below rate)
  → Always ~100 tokens available (bucket stays full)
  → All requests ALLOWED
  → No denials
```

**Scenario 2: Burst then quiet**
```
10:00:00 - User sends 100 rapid requests
  → All ALLOWED (consume all 100 tokens)
  → Bucket: 0 tokens

10:00:00.001 - Request 101
  → DENIED (no tokens, retry_after = 0.6s)

10:00:01 - 1 second passed
  → Refilled: 1.67 tokens
  → 1 request ALLOWED

10:00:02 - 1 second passed
  → Refilled: 1.67 tokens
  → 1 request ALLOWED

10:00:60 - 60 seconds total
  → Bucket full again: 100 tokens
```

**Scenario 3: Sustained load**
```
User sends 2 requests/second (above rate of 1.67/sec)
  → First 100 requests: ALLOWED (burst capacity)
  → Bucket drains faster than refills
  → After bucket empty: 1-2 requests/sec ALLOWED, rest DENIED
  → Average settles to refill rate (1.67/sec)
```

### Pros & Cons

**✅ Pros:**
- **Natural burst handling** - Explicit burst capacity design
- **Smooth rate limiting** - Industry standard approach
- **Intuitive** - Easy to explain to users ("bucket of tokens")
- **Flexible** - Works well with variable traffic
- **Standard** - Used by AWS, Google Cloud, most SaaS APIs
- **Good UX** - Allows occasional bursts without penalties

**❌ Cons:**
- **Complex implementation** - Refill logic, float arithmetic
- **Float precision** - Potential rounding errors at scale
- **Harder to debug** - State includes timestamp and fractional tokens
- **Clock drift** - Timestamp-dependent, affected by clock changes
- **Harder to reason about** - "How many requests can I make right now?" depends on history

### Use Cases

1. **Public APIs**
   ```go
   // Standard API rate limiting
   config := &Config{
       Algorithm: TokenBucket,
       Limit:     1000,  // Burst capacity
       Window:    time.Minute,  // 1000/min = ~16.67/sec sustained
   }
   ```
   - Users expect burst tolerance
   - Industry standard

2. **Variable Workloads**
   ```go
   // Email sending (batch in morning, quiet rest of day)
   config := &Config{
       Algorithm: TokenBucket,
       Limit:     1000,
       Window:    time.Hour,
   }
   ```
   - Legitimate usage patterns include bursts

3. **Media Streaming**
   - Burst for quality changes
   - Smooth rate overall
   - Buffering-friendly

4. **CDN/Proxy**
   - Handle traffic spikes gracefully
   - Prevent abuse while allowing bursts

### Performance Characteristics

**Throughput:** ~35,000 RPS (single Redis instance)
**Latency:** 0.7-1.2ms (Redis script + float math)
**Memory:** ~150 bytes per active user

---

## Algorithm Evolution in Production

Most systems evolve through these stages:

```
Stage 1: Proof of Concept
└─> Fixed Window (easy to implement, test, understand)
    - Get something working quickly
    - Validate rate limiting helps
    - Learn usage patterns

Stage 2: Early Production
└─> Token Bucket (better UX, industry standard)
    - Improve user experience
    - Handle variable traffic better
    - Standard approach for APIs

Stage 3: Scale/Monetization
└─> Sliding Window for paid tiers (precision matters)
    Token Bucket for free tier (good UX)
    - Different algorithms for different needs
    - Paid users get strict, fair limits
    - Free users get burst tolerance
```

## Choosing the Right Algorithm

### Decision Matrix

| Requirement | Recommended Algorithm |
|-------------|----------------------|
| Simplest possible | Fixed Window |
| Fastest performance | Fixed Window |
| Lowest memory usage | Fixed Window |
| Prevent boundary gaming | Sliding Window or Token Bucket |
| Strict SLA enforcement | Sliding Window |
| Money/billing involved | Sliding Window |
| API rate limiting | Token Bucket |
| Variable traffic patterns | Token Bucket |
| Burst tolerance needed | Token Bucket |
| Industry standard approach | Token Bucket |

### Common Combinations

**Multi-tier Application:**
```go
// Different algorithms for different use cases
loginLimiter := NewFixedWindow(...)      // Simple, fast
quotaLimiter := NewSlidingWindow(...)    // Precise, fair
apiLimiter := NewTokenBucket(...)        // Standard, flexible
```

## References

- [GCRA Algorithm (Generic Cell Rate Algorithm)](https://en.wikipedia.org/wiki/Generic_cell_rate_algorithm)
- [Stripe Rate Limiting](https://stripe.com/blog/rate-limiters)
- [Cloudflare Rate Limiting](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)
- [Kong Rate Limiting](https://docs.konghq.com/hub/kong-inc/rate-limiting/)
- [AWS API Gateway Throttling](https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-request-throttling.html)
