package services

// twenty_scheduler.go — background scheduler for Twenty product sync.
//
// Mirrors the EventoryScheduler pattern. Call GetTwentyScheduler().Reset()
// at startup (and after saving new settings) to start/restart the ticker.
// Call GetTwentyScheduler().RunOnce() to trigger an immediate sync without
// waiting for the next tick.

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// TwentyScheduler runs a background goroutine that periodically syncs
// WarehouseCore products to the Twenty CRM. Singleton; call GetTwentyScheduler().
type TwentyScheduler struct {
	mu       sync.Mutex
	resetMu  sync.Mutex
	stopCh   chan struct{}
	stopped  bool
	tickerWg sync.WaitGroup
	wg       sync.WaitGroup
	running  int32

	// syncFn is called on each tick and on RunOnce. Overridable in tests.
	syncFn func()
	// configFn reads the current config. Overridable in tests.
	configFn func() (*TwentyConfig, error)
}

var (
	globalTwentySchedulerOnce sync.Once
	globalTwentyScheduler     *TwentyScheduler
)

var allowedTwentySyncIntervals = map[int]bool{
	15:   true,
	30:   true,
	60:   true,
	120:  true,
	240:  true,
	480:  true,
	1440: true,
}

// IsAllowedTwentySyncInterval returns true when n is a valid non-zero sync
// interval in minutes for the Twenty scheduler.
func IsAllowedTwentySyncInterval(n int) bool {
	return allowedTwentySyncIntervals[n]
}

// GetTwentyScheduler returns the singleton scheduler.
func GetTwentyScheduler() *TwentyScheduler {
	globalTwentySchedulerOnce.Do(func() {
		globalTwentyScheduler = &TwentyScheduler{
			syncFn:   defaultTwentySync,
			configFn: GetTwentyConfig,
		}
	})
	return globalTwentyScheduler
}

// TryAcquireSync marks a sync in progress. Returns false if one is already running.
func (s *TwentyScheduler) TryAcquireSync() bool {
	return atomic.CompareAndSwapInt32(&s.running, 0, 1)
}

// ReleaseSync clears the in-progress flag.
func (s *TwentyScheduler) ReleaseSync() {
	atomic.StoreInt32(&s.running, 0)
}

// RunOnce triggers an immediate sync in the background (non-blocking).
// It is safe to call from any goroutine.
func (s *TwentyScheduler) RunOnce() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TWENTY SCHEDULER] panic in sync: %v", r)
			}
		}()
		if !s.TryAcquireSync() {
			log.Printf("[TWENTY SCHEDULER] sync already running, skipping")
			return
		}
		defer s.ReleaseSync()
		s.syncFn()
	}()
}

// Reset stops any existing ticker and starts a new one based on the current
// sync_interval_minutes setting. A RunOnce initial sync fires immediately
// regardless of the interval. Safe to call from any goroutine.
func (s *TwentyScheduler) Reset() {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()

	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.mu.Unlock()

	s.tickerWg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	cfg, err := s.configFn()
	if err != nil {
		log.Printf("[TWENTY SCHEDULER] failed to read config: %v", err)
		return
	}

	intervalMins := cfg.SyncIntervalMinutes

	// Always run an initial sync when Reset is called (e.g. at startup).
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TWENTY SCHEDULER] panic in initial sync: %v", r)
			}
		}()
		if !s.TryAcquireSync() {
			return
		}
		defer s.ReleaseSync()
		s.syncFn()
	}()

	if intervalMins <= 0 {
		log.Printf("[TWENTY SCHEDULER] automatic sync disabled (interval=0)")
		return
	}
	if !IsAllowedTwentySyncInterval(intervalMins) {
		log.Printf("[TWENTY SCHEDULER] unsupported sync_interval_minutes=%d, disabling automatic sync", intervalMins)
		return
	}

	duration := time.Duration(intervalMins) * time.Minute
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	s.tickerWg.Add(1)

	go func() {
		defer s.tickerWg.Done()
		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		log.Printf("[TWENTY SCHEDULER] started, interval=%dm", intervalMins)
		for {
			select {
			case <-ticker.C:
				if !s.TryAcquireSync() {
					log.Printf("[TWENTY SCHEDULER] previous sync still running, skipping tick")
					continue
				}
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					s.ReleaseSync()
					return
				}
				s.wg.Add(1)
				s.mu.Unlock()
				go func() {
					defer s.wg.Done()
					defer s.ReleaseSync()
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[TWENTY SCHEDULER] panic in scheduled sync: %v", r)
						}
					}()
					s.syncFn()
				}()
			case <-stopCh:
				log.Printf("[TWENTY SCHEDULER] ticker stopped")
				return
			}
		}
	}()
}

// Stop shuts down the scheduler and waits for any in-progress sync to finish.
func (s *TwentyScheduler) Stop() {
	s.mu.Lock()
	s.stopped = true
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.mu.Unlock()
	s.tickerWg.Wait()
	s.wg.Wait()
}

// defaultTwentySync is the production sync function: reads config and calls
// the product sync logic in the handlers package via a registered hook.
func defaultTwentySync() {
	cfg, err := GetTwentyConfig()
	if err != nil {
		log.Printf("[TWENTY SCHEDULER] failed to load config: %v", err)
		return
	}
	if cfg.BaseURL == "" {
		log.Printf("[TWENTY SCHEDULER] base_url not configured, skipping")
		return
	}
	if cfg.APIKey == "" {
		log.Printf("[TWENTY SCHEDULER] api_key not configured, skipping")
		return
	}

	if TwentyProductSyncHook != nil {
		created, updated, err := TwentyProductSyncHook()
		if err != nil {
			log.Printf("[TWENTY SCHEDULER] product sync failed: %v", err)
			return
		}
		log.Printf("[TWENTY SCHEDULER] product sync complete: %d created, %d updated", created, updated)
	}
}

// TwentyProductSyncHook is set by the handlers package at startup to avoid an
// import cycle (handlers → services → handlers).
var TwentyProductSyncHook func() (int, int, error)
