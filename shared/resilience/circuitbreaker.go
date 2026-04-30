package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
	IsFailure        func(error) bool
	Now              func() time.Time
}

type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	successThreshold int
	openTimeout      time.Duration
	isFailure        func(error) bool
	now              func() time.Time
	state            CircuitState
	failures         int
	successes        int
	openedAt         time.Time
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	failureThreshold := config.FailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = 5
	}

	successThreshold := config.SuccessThreshold
	if successThreshold <= 0 {
		successThreshold = 1
	}

	openTimeout := config.OpenTimeout
	if openTimeout <= 0 {
		openTimeout = 30 * time.Second
	}

	isFailure := config.IsFailure
	if isFailure == nil {
		isFailure = func(err error) bool { return err != nil }
	}

	now := config.Now
	if now == nil {
		now = time.Now
	}

	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		openTimeout:      openTimeout,
		isFailure:        isFailure,
		now:              now,
		state:            CircuitClosed,
	}
}

func (b *CircuitBreaker) State() CircuitState {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.transitionOpenToHalfOpenLocked(b.now())
	return b.state
}

func (b *CircuitBreaker) Do(ctx context.Context, operation func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := b.beforeCall(); err != nil {
		return err
	}

	err := operation(ctx)
	b.afterCall(err)
	return err
}

func (b *CircuitBreaker) beforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.now()
	b.transitionOpenToHalfOpenLocked(now)
	if b.state == CircuitOpen {
		return ErrCircuitOpen
	}
	return nil
}

func (b *CircuitBreaker) afterCall(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil && b.isFailure(err) {
		b.recordFailureLocked()
		return
	}
	if err != nil {
		return
	}
	b.recordSuccessLocked()
}

func (b *CircuitBreaker) recordFailureLocked() {
	b.successes = 0
	b.failures++
	if b.state == CircuitHalfOpen || b.failures >= b.failureThreshold {
		b.state = CircuitOpen
		b.openedAt = b.now()
		b.failures = 0
	}
}

func (b *CircuitBreaker) recordSuccessLocked() {
	b.failures = 0
	if b.state != CircuitHalfOpen {
		return
	}

	b.successes++
	if b.successes >= b.successThreshold {
		b.state = CircuitClosed
		b.successes = 0
	}
}

func (b *CircuitBreaker) transitionOpenToHalfOpenLocked(now time.Time) {
	if b.state == CircuitOpen && !now.Before(b.openedAt.Add(b.openTimeout)) {
		b.state = CircuitHalfOpen
		b.successes = 0
	}
}
