package debounce

import (
	"time"
)

type Option func(*Config)

func Leading() Option {
	return func(c *Config) {
		c.Leading = true
	}
}

func Trailing() Option {
	return func(c *Config) {
		c.Trailing = true
	}
}

func MaxWait(maxWait time.Duration) Option {
	return func(c *Config) {
		c.MaxWait = maxWait
	}
}
