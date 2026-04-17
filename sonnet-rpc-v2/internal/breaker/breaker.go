package breaker

import (
	"log"
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	mu sync.Mutex

	state State

	// Statistics
	failureCount int
	successCount int

	// Configuration parameters
	windowSize       int           // Statistics window size (count)
	failureThreshold float64       // Failure rate threshold
	openTimeout      time.Duration // Circuit breaker open duration

	// State control
	lastStateChange time.Time
	halfOpenProbe   bool // Whether there is a probe request in half-open state
}

func NewCircuitBreaker(windowSize int, failureThreshold float64, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            Closed,
		windowSize:       windowSize,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
		lastStateChange:  time.Now(),
	}
}
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		return true

	case Open:
		// Circuit breaker duration elapsed, enter half-open
		if time.Since(cb.lastStateChange) > cb.openTimeout {
			cb.state = HalfOpen
			cb.halfOpenProbe = false
			return true
		}
		return false

	case HalfOpen:
		// Only one probe request allowed
		if cb.halfOpenProbe {
			return false
		}
		cb.halfOpenProbe = true
		return true
	}

	return true
}
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		cb.successCount++

	case HalfOpen:
		// Probe succeeded -> recover
		cb.toClosed()

	case Open:
		// Theoretically should not enter this block
		log.Println("Theoretically untriggered")
	}
}
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		cb.failureCount++

		total := cb.failureCount + cb.successCount
		if total < cb.windowSize {
			return
		}

		rate := float64(cb.failureCount) / float64(total)
		if rate >= cb.failureThreshold {
			cb.toOpen()
			return
		}

		cb.resetCounts()

	case HalfOpen:
		// Probe failed -> open circuit breaker again
		cb.toOpen()

	case Open:
		// Already open, ignore
	}
}
func (cb *CircuitBreaker) toOpen() {
	cb.state = Open
	cb.lastStateChange = time.Now()
	cb.resetCounts()
	cb.halfOpenProbe = false
}

func (cb *CircuitBreaker) toClosed() {
	cb.state = Closed
	cb.lastStateChange = time.Now()
	cb.resetCounts()
	cb.halfOpenProbe = false
}
func (cb *CircuitBreaker) resetCounts() {
	cb.failureCount = 0
	cb.successCount = 0
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
