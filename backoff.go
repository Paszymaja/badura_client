package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type BackoffConfig struct {
	MinBackoff time.Duration // start backoff at this level - 100*time.Millisecond
	MaxBackoff time.Duration // increase exponentially to this level - 5*time.Second
	MaxRetries int           // give up after this many; zero means infinite retries - 3
}

type Backoff struct {
	cfg          BackoffConfig
	ctx          context.Context
	numRetries   int
	nextDelayMin time.Duration
	nextDelayMax time.Duration
}

func NewBackoff(ctx context.Context, cfg BackoffConfig) *Backoff {
	return &Backoff{
		cfg:          cfg,
		ctx:          ctx,
		nextDelayMin: cfg.MinBackoff,
		nextDelayMax: doubleDuration(cfg.MinBackoff, cfg.MaxBackoff),
	}
}

func (b *Backoff) Reset() {
	b.numRetries = 0
	b.nextDelayMin = b.cfg.MinBackoff
	b.nextDelayMax = doubleDuration(b.cfg.MinBackoff, b.cfg.MaxBackoff)
}

func (b *Backoff) Ongoing() bool {
	// Stop if Context has errored or max retry count is exceeded
	return b.ctx.Err() == nil && (b.cfg.MaxRetries == 0 || b.numRetries < b.cfg.MaxRetries)
}

func (b *Backoff) Err() error {
	if b.ctx.Err() != nil {
		return b.ctx.Err()
	}
	if b.cfg.MaxRetries != 0 && b.numRetries >= b.cfg.MaxRetries {
		return fmt.Errorf("terminated after %d retries", b.numRetries)
	}
	return nil
}

func (b *Backoff) Wait() {
	sleepTime := b.IncAndGetNextDelay()

	if b.Ongoing() {
		select {
		case <-b.ctx.Done():
		case <-time.After(sleepTime):
		}
	}
}

func (b *Backoff) IncAndGetNextDelay() time.Duration {
	b.numRetries++

	// Handle the edge case the min and max have the same value
	if b.nextDelayMin >= b.nextDelayMax {
		return b.nextDelayMin
	}

	// Add a jitter within the next exponential backoff range
	sleepTime := b.nextDelayMin + time.Duration(rand.Int63n(int64(b.nextDelayMax-b.nextDelayMin)))

	// Apply the exponential backoff to calculate the next jitter
	// range, unless we've already reached the max
	if b.nextDelayMax < b.cfg.MaxBackoff {
		b.nextDelayMin = doubleDuration(b.nextDelayMin, b.cfg.MaxBackoff)
		b.nextDelayMax = doubleDuration(b.nextDelayMax, b.cfg.MaxBackoff)
	}

	return sleepTime
}

func doubleDuration(current time.Duration, max time.Duration) time.Duration {
	current = current * 2

	if current <= max {
		return current
	}

	return max
}
