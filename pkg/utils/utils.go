package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func SafeConvertString(b []byte) string {
	bld := strings.Builder{}
	bld.Grow(len(b))
	for _, c := range string(b) {
		if c == '\uFFFD' {
			bld.WriteRune('.')
		} else {
			bld.WriteRune(c)
		}
	}
	return bld.String()
}

func LimitLog(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return "(log truncated)\n" + s[len(s)-limit:]
}

func WithRetry(ctx context.Context, maxRetryCount int, fn func(ctx context.Context) error) error {
	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 60 * time.Second
	)

	var err error
	backoff := initialBackoff
	for i := 0; i < maxRetryCount; i++ {
		err = fn(ctx)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("Encountered error, retrying in %v: %v\n", backoff, err)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}

		backoff = min(backoff*2, maxBackoff)
	}

	return fmt.Errorf("max retry count %d reached: %w", maxRetryCount, err)
}
