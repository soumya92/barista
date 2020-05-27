package timing

import (
	"sync"
	"time"
)

var _ schedulerImpl = &timeScheduler{}

// timeScheduler is a scheduler backed by "time" package.
type timeScheduler struct {
	mu      sync.Mutex
	timer   *time.Timer
	ticker  *time.Ticker
	quitter chan struct{}
}

// NewScheduler creates a new scheduler.
//
// The scheduler is backed by "time" package. Its "At" implementation
// is unreliable, as it's unable to take system suspend and time adjustments
// into account.
func NewScheduler() *Scheduler {
	if testModeScheduler := maybeNewTestModeScheduler(); testModeScheduler != nil {
		return newScheduler(testModeScheduler)
	}
	return newScheduler(&timeScheduler{})
}

// At implements the schedulerImpl interface.
func (s *timeScheduler) At(when time.Time, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.timer = time.AfterFunc(when.Sub(Now()), f)
}

// After implements the schedulerImpl interface.
func (s *timeScheduler) After(delay time.Duration, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.timer = time.AfterFunc(delay, f)
}

// Every implements the schedulerImpl interface.
func (s *timeScheduler) Every(interval time.Duration, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.quitter = make(chan struct{})
	s.ticker = time.NewTicker(interval)
	go func() {
		s.mu.Lock()
		ticker := s.ticker
		quitter := s.quitter
		s.mu.Unlock()
		if ticker == nil || quitter == nil {
			// Scheduler stopped before goroutine was started.
			return
		}
		for {
			select {
			case <-ticker.C:
				f()
			case <-quitter:
				return
			}
		}
	}()
}

// EveryAlign implements the schedulerImpl interface.
func (s *timeScheduler) EveryAlign(interval time.Duration, offset time.Duration, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	quitter := make(chan struct{})
	s.quitter = quitter
	go func() {
		var timer *time.Timer
		for {
			now := time.Now()
			next := nextAlignedExpiration(now, interval, offset)
			delay := next.Sub(now)
			if timer == nil {
				timer = time.NewTimer(delay)
				defer timer.Stop()
			} else {
				timer.Reset(delay)
			}
			select {
			case <-timer.C:
				f()
			case <-quitter:
				return
			}
		}
	}()
}

// Stop implements the schedulerImpl interface.
func (s *timeScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
}

// Close implements the schedulerImpl interface.
func (s *timeScheduler) Close() {
	s.Stop()
}

func (s *timeScheduler) stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	if s.quitter != nil {
		close(s.quitter)
		s.quitter = nil
	}
}
