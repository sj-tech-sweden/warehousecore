package services

import (
	"sync/atomic"
	"testing"
	"time"
)

func newTestTwentyScheduler(syncFn func(), interval int) *TwentyScheduler {
	if syncFn == nil {
		syncFn = func() {}
	}
	return &TwentyScheduler{
		syncFn: syncFn,
		configFn: func() (*TwentyConfig, error) {
			return &TwentyConfig{SyncIntervalMinutes: interval}, nil
		},
	}
}

func TestTwentySchedulerReset_DisablesAutomaticSyncForUnsupportedInterval(t *testing.T) {
	var syncCalls int32
	s := newTestTwentyScheduler(func() {
		atomic.AddInt32(&syncCalls, 1)
	}, int(^uint(0)>>1))

	s.Reset()
	s.wg.Wait()
	defer s.Stop()

	if got := atomic.LoadInt32(&syncCalls); got != 1 {
		t.Fatalf("expected exactly one initial sync for unsupported interval, got %d", got)
	}
	if s.stopCh != nil {
		t.Fatal("expected no ticker to be started for unsupported interval")
	}
}

func TestTwentySchedulerReset_StartsTickerForAllowedInterval(t *testing.T) {
	s := newTestTwentyScheduler(nil, 15)

	s.Reset()
	defer s.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		started := s.stopCh != nil
		s.mu.Unlock()
		if started {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("expected ticker to be started for allowed interval")
}
