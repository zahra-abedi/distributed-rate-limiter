package ratelimiter

import "errors"

var (
    // ErrInvalidConfig indicates the configuration is invalid
    ErrInvalidConfig = errors.New("invalid rate limiter configuration")

    // ErrStorageUnavailable indicates the storage backend (Redis) is unavailable
    ErrStorageUnavailable = errors.New("rate limiter storage unavailable")

    // ErrInvalidKey indicates the provided key is invalid (e.g., empty)
    ErrInvalidKey = errors.New("invalid key: must not be empty")

    // ErrInvalidN indicates the N parameter for AllowN is invalid
    ErrInvalidN = errors.New("invalid n: must be greater than 0")

    // ErrClosed indicates the rate limiter has been closed
    ErrClosed = errors.New("rate limiter is closed")
)
