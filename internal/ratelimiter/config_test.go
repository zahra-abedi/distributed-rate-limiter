package ratelimiter

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "valid token bucket config",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
			},
			wantErr: false,
		},
		{
			name: "valid sliding window config",
			config: &Config{
				Algorithm: SlidingWindow,
				Limit:     1000,
				Window:    time.Hour,
			},
			wantErr: false,
		},
		{
			name: "valid fixed window config",
			config: &Config{
				Algorithm: FixedWindow,
				Limit:     50,
				Window:    time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing algorithm",
			config: &Config{
				Limit:  100,
				Window: time.Minute,
			},
			wantErr: true,
			errMsg:  "algorithm is required",
		},
		{
			name: "invalid algorithm",
			config: &Config{
				Algorithm: "invalid_algo",
				Limit:     100,
				Window:    time.Minute,
			},
			wantErr: true,
			errMsg:  "unknown algorithm",
		},
		{
			name: "zero limit",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     0,
				Window:    time.Minute,
			},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name: "negative limit",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     -10,
				Window:    time.Minute,
			},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name: "zero window",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    0,
			},
			wantErr: true,
			errMsg:  "window must be greater than 0",
		},
		{
			name: "negative window",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "window must be greater than 0",
		},
		{
			name: "window too small",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    500 * time.Nanosecond,
			},
			wantErr: true,
			errMsg:  "window too small",
		},
		{
			name: "window too large",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    400 * 24 * time.Hour, // Over 1 year
			},
			wantErr: true,
			errMsg:  "window too large",
		},
		{
			name: "valid with custom prefix",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    "api",
			},
			wantErr: false,
		},
		{
			name: "valid with empty prefix",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    "",
			},
			wantErr: false,
		},
		{
			name: "valid with fail-open",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				FailOpen:  true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   *Config
	}{
		{
			name:   "nil config",
			config: nil,
			want:   nil,
		},
		{
			name: "config without prefix gets default",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
			},
			want: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    DefaultPrefix,
			},
		},
		{
			name: "config with custom prefix unchanged",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    "api",
			},
			want: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    "api",
			},
		},
		{
			name: "config with empty prefix gets default",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    "",
			},
			want: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    DefaultPrefix,
			},
		},
		{
			name: "fail-open preserved",
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				FailOpen:  true,
			},
			want: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
				Prefix:    DefaultPrefix,
				FailOpen:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.WithDefaults()

			if (got == nil) != (tt.want == nil) {
				t.Errorf("WithDefaults() = %v, want %v", got, tt.want)
				return
			}

			if got != nil && tt.want != nil {
				if got.Algorithm != tt.want.Algorithm {
					t.Errorf("Algorithm = %v, want %v", got.Algorithm, tt.want.Algorithm)
				}
				if got.Limit != tt.want.Limit {
					t.Errorf("Limit = %v, want %v", got.Limit, tt.want.Limit)
				}
				if got.Window != tt.want.Window {
					t.Errorf("Window = %v, want %v", got.Window, tt.want.Window)
				}
				if got.Prefix != tt.want.Prefix {
					t.Errorf("Prefix = %v, want %v", got.Prefix, tt.want.Prefix)
				}
				if got.FailOpen != tt.want.FailOpen {
					t.Errorf("FailOpen = %v, want %v", got.FailOpen, tt.want.FailOpen)
				}
			}
		})
	}
}

func TestConfig_FormatKey(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		key    string
		want   string
	}{
		{
			name: "with default prefix",
			config: &Config{
				Prefix: DefaultPrefix,
			},
			key:  "user:123",
			want: "ratelimit:user:123",
		},
		{
			name: "with custom prefix",
			config: &Config{
				Prefix: "api",
			},
			key:  "user:123",
			want: "api:user:123",
		},
		{
			name: "with empty prefix",
			config: &Config{
				Prefix: "",
			},
			key:  "user:123",
			want: "user:123",
		},
		{
			name:   "nil config uses default",
			config: nil,
			key:    "user:123",
			want:   "ratelimit:user:123",
		},
		{
			name: "complex key with multiple colons",
			config: &Config{
				Prefix: "app",
			},
			key:  "tenant:abc:user:xyz:resource:file",
			want: "app:tenant:abc:user:xyz:resource:file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.FormatKey(tt.key)
			if got != tt.want {
				t.Errorf("FormatKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_KeyPrefix(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name:   "nil config returns default",
			config: nil,
			want:   DefaultPrefix,
		},
		{
			name: "config with prefix",
			config: &Config{
				Prefix: "api",
			},
			want: "api",
		},
		{
			name: "config with empty prefix",
			config: &Config{
				Prefix: "",
			},
			want: "",
		},
		{
			name: "config without prefix field set",
			config: &Config{
				Algorithm: TokenBucket,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.KeyPrefix()
			if got != tt.want {
				t.Errorf("KeyPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
