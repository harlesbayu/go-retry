package goretry

import (
	"context"
	"time"

	pkgRetry "github.com/sethvargo/go-retry"
)

type BackoffType string

const (
	maxRetries   int         = 3
	initialDelay             = 3 * time.Second
	maxDuration              = 10 * time.Second
	jitter                   = 200 * time.Millisecond
	Fibonacci    BackoffType = "fibonacci"
	Constant     BackoffType = "constant"
	Exponential  BackoffType = "exponential"
)

type Config struct {
	InitialDelay time.Duration
	MaxRetries   int
	BackoffType  BackoffType
	Jitter       time.Duration
	MaxDuration  time.Duration
}

/*
DefaultConfig initialize the default configuration
  - InitialDelay: default "3s"
  - MaxRetries: default "3"
  - BackoffType: default "constant"
  - MaxDuration: default "10s"
  - Jitter: default "2.5s"

Notes:
  - MaxDuration is used to set the maximum total amount of time that backoff should execute. List of BackoffType "fibonacci", "constant", "exponential"
  - Jitter is used to to reduce the changes of a thundering herd, add random jitter to the returned value
  - To use infinity retry, set MaxDuration to "0s" and MaxRetries to "-1"
  - To disable jitter, set jitter to "0s"
*/
func DefaultConfig() Config {
	return Config{
		InitialDelay: initialDelay,
		MaxRetries:   maxRetries,
		BackoffType:  Constant,
		MaxDuration:  maxDuration,
		Jitter:       jitter,
	}
}

// UpdateConfig updates the provided values without changing the existing configuration
func (c *Config) UpdateConfig(newConfig Config) {
	if newConfig.InitialDelay != 0 {
		c.InitialDelay = newConfig.InitialDelay
	}
	if newConfig.MaxRetries != 0 {
		c.MaxRetries = newConfig.MaxRetries
	}
	if newConfig.BackoffType != "" {
		c.BackoffType = newConfig.BackoffType
	}
	if newConfig.Jitter != 0 {
		c.Jitter = newConfig.Jitter
	}
	if newConfig.MaxDuration != 0 {
		c.MaxDuration = newConfig.MaxDuration
	}
}

// DoRetry will perform a retry by entering a list of errors that need to be retried
func DoRetry(ctx context.Context, cfg Config, fn func(context.Context) error, retryableError []error) error {
	b := getBackoff(cfg)

	fn2 := func() func(ctx context.Context) error {
		return func(ctx context.Context) error {
			err := fn(ctx)

			if err == nil {
				return nil
			}

			if len(retryableError) > 0 {
				for _, v := range retryableError {
					if err.Error() == v.Error() {
						err = pkgRetry.RetryableError(v)
					}
				}
			}

			return err
		}
	}

	return pkgRetry.Do(ctx, b, fn2())
}

// DoRetryWithCustomRetryableError will perform a retry by implementing **RetryableError** on the error to be retried
func DoRetryWithCustomRetryableError(ctx context.Context, cfg Config, fn pkgRetry.RetryFunc) error {
	b := getBackoff(cfg)
	err := pkgRetry.Do(ctx, b, fn)

	return err
}

// RetryableError marks an error as retryable
func RetryableError(err error) error {
	return pkgRetry.RetryableError(err)
}

// Set config backoff
func getBackoff(cfg Config) pkgRetry.Backoff {
	var b pkgRetry.Backoff
	switch cfg.BackoffType {
	case Constant:
		b = pkgRetry.NewConstant(cfg.InitialDelay)
	case Exponential:
		b = pkgRetry.NewExponential(cfg.InitialDelay)
	case Fibonacci:
		b = pkgRetry.NewFibonacci(cfg.InitialDelay)
	default:
		b = pkgRetry.NewExponential(cfg.InitialDelay)
	}

	if cfg.Jitter > 0 {
		b = pkgRetry.WithJitter(cfg.Jitter, b)
	}

	if cfg.MaxDuration > 0 {
		b = pkgRetry.WithMaxDuration(cfg.MaxDuration, b)
	}

	if cfg.MaxRetries > 0 {
		b = pkgRetry.WithMaxRetries(uint64(cfg.MaxRetries), b)
	} else if cfg.MaxRetries == 0 {
		b = pkgRetry.WithMaxRetries(uint64(maxRetries), b)
	}

	return b
}
