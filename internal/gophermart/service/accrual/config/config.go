package config

import "time"

const (
	DefaultAddr           = "http://localhost:8081"
	DefaultRetryAfter     = 60 * time.Second
	DefaultRequestTimeout = 30 * time.Second
)

type Config struct {
	RetryAfterDefault time.Duration
	RequestTimeout    time.Duration
	Addr              string
}
