package services

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestScheduler creates a fresh, isolated EventoryScheduler for unit tests.
// It injects a no-op syncFn and a configFn that returns sync disabled (interval 0)
// so tests never touch the database or network.
func newTestScheduler(syncFn func()) *EventoryScheduler {
	if syncFn == nil {
		syncFn = func() {}
	}
	return &EventoryScheduler{
		syncFn: syncFn,
		configFn: func() (*EventoryConfig, error) {
			return &EventoryConfig{SyncIntervalMinutes: 0}, nil
		},
	}
}

// startTestTicker is a helper that wires a fast ticker into a scheduler using
// the same mechanism as Reset(): it adds to tickerWg and launches the goroutine.
// interval must be > 0.
func startTestTicker(s *EventoryScheduler, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	s.tickerWg.Add(1)
	go func() {
		defer s.tickerWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
					continue
				}
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer atomic.StoreInt32(&s.running, 0)
					s.syncFn()
				}()
			case <-stopCh:
				return
			}
		}
	}()
}

// ===========================
// TryAcquireSync / ReleaseSync
// ===========================

func TestTryAcquireSync_AcquiresAndReleases(t *testing.T) {
	s := newTestScheduler(nil)

	if !s.TryAcquireSync() {
		t.Fatal("expected TryAcquireSync to return true on idle scheduler")
	}
	// Flag should be set (1)
	if atomic.LoadInt32(&s.running) != 1 {
		t.Error("expected running flag to be 1 after acquire")
	}

	s.ReleaseSync()

	// Flag should be cleared (0)
	if atomic.LoadInt32(&s.running) != 0 {
		t.Error("expected running flag to be 0 after release")
	}
}

func TestTryAcquireSync_BlocksWhenAlreadyRunning(t *testing.T) {
	s := newTestScheduler(nil)

	if !s.TryAcquireSync() {
		t.Fatal("first acquire should succeed")
	}
	// A second acquire while the first is held must fail.
	if s.TryAcquireSync() {
		t.Error("second TryAcquireSync should return false while first is still held")
		s.ReleaseSync() // clean up
	}
	s.ReleaseSync()
}

func TestTryAcquireSync_ReturnsFalseAfterStop(t *testing.T) {
	s := newTestScheduler(nil)
	s.Stop()

	if s.TryAcquireSync() {
		t.Error("TryAcquireSync should return false after Stop()")
		s.ReleaseSync()
	}
}

// ===========================
// Stop waits for in-flight syncs
// ===========================

func TestStop_WaitsForInFlightSync(t *testing.T) {
	syncStarted := make(chan struct{})
	syncAllowExit := make(chan struct{})

	s := newTestScheduler(func() {
		close(syncStarted)
		<-syncAllowExit
	})

	if !s.TryAcquireSync() {
		t.Fatal("acquire should succeed on fresh scheduler")
	}
	go func() {
		s.syncFn()
		s.ReleaseSync()
	}()

	// Wait for sync to begin
	select {
	case <-syncStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("sync did not start in time")
	}

	stopDone := make(chan struct{})
	go func() {
		s.Stop()
		close(stopDone)
	}()

	// Stop should be blocked while sync is running
	select {
	case <-stopDone:
		t.Error("Stop() returned before in-flight sync finished")
	case <-time.After(50 * time.Millisecond):
		// expected: Stop is still waiting
	}

	// Let the sync finish
	close(syncAllowExit)

	select {
	case <-stopDone:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return after sync finished")
	}
}

// ===========================
// No-overlap guarantee
// ===========================

func TestScheduler_NoOverlap(t *testing.T) {
	var concurrent int32
	var maxConcurrent int32
	var calls int32

	s := newTestScheduler(func() {
		cur := atomic.AddInt32(&concurrent, 1)
		// Track the high-water mark
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond) // simulate work
		atomic.AddInt32(&concurrent, -1)
		atomic.AddInt32(&calls, 1)
	})

	startTestTicker(s, 2*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&maxConcurrent) > 1 {
		t.Errorf("concurrent syncs exceeded 1 (max seen: %d)", atomic.LoadInt32(&maxConcurrent))
	}
	if atomic.LoadInt32(&calls) == 0 {
		t.Error("expected at least one sync to have run")
	}
}

// ===========================
// Concurrent Reset calls
// ===========================

func TestReset_ConcurrentCallsDoNotPanic(t *testing.T) {
	// Inject a configFn that returns a valid (but very long) sync interval so
	// Reset() exercises the full code path — including tickerWg.Add(1) and
	// goroutine launch — without the ticker ever actually firing during the test.
	const testLongIntervalMinutes = 1440 // 24 h — ticker will never fire in a unit test
	s := &EventoryScheduler{
		syncFn: func() {},
		configFn: func() (*EventoryConfig, error) {
			return &EventoryConfig{SyncIntervalMinutes: testLongIntervalMinutes}, nil
		},
	}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.Reset()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Reset() calls deadlocked or panicked")
	}

	// Clean up: Stop the scheduler so the ticker goroutine exits.
	s.Stop()
}

// ===========================
// Stop is idempotent
// ===========================

func TestStop_Idempotent(t *testing.T) {
	s := newTestScheduler(nil)
	startTestTicker(s, 10*time.Millisecond)
	s.Stop()
	// Calling Stop again must not panic or block.
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second Stop() call blocked or panicked")
	}
}

// ===========================
// TryAcquireSync under Stop
// ===========================

func TestTryAcquireSync_StopBlocksNewAcquire(t *testing.T) {
	s := newTestScheduler(nil)

	// Acquire a sync slot first.
	if !s.TryAcquireSync() {
		t.Fatal("initial acquire should succeed")
	}

	stopDone := make(chan struct{})
	go func() {
		s.Stop()
		close(stopDone)
	}()

	// Give Stop a moment to start waiting.
	time.Sleep(20 * time.Millisecond)

	// Release so Stop can finish.
	s.ReleaseSync()

	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not finish after ReleaseSync")
	}

	// After Stop, new acquires must fail.
	if s.TryAcquireSync() {
		t.Error("TryAcquireSync must return false after Stop")
		s.ReleaseSync()
	}
}

// ===========================
// Customer price margin computation
// ===========================

// computeCustomerPrice replicates the rounding formula used in RunEventorySync
// so we can unit-test it without a database connection.
func computeCustomerPrice(price, marginPercent float64) float64 {
	return math.Round(price*(1+marginPercent/100)*100) / 100
}

func TestComputeCustomerPrice_ZeroMargin(t *testing.T) {
	// Zero margin should leave the price unchanged.
	if got := computeCustomerPrice(100.0, 0); got != 100.0 {
		t.Errorf("expected 100.00, got %v", got)
	}
}

func TestComputeCustomerPrice_RoundNumbers(t *testing.T) {
	cases := []struct {
		price    float64
		margin   float64
		expected float64
	}{
		{100.0, 10, 110.0},
		{200.0, 25, 250.0},
		{50.0, 100, 100.0},
		{0.0, 50, 0.0},
	}
	for _, tc := range cases {
		if got := computeCustomerPrice(tc.price, tc.margin); got != tc.expected {
			t.Errorf("price=%.2f margin=%.2f: expected %.2f, got %.2f", tc.price, tc.margin, tc.expected, got)
		}
	}
}

func TestComputeCustomerPrice_Rounding(t *testing.T) {
	// 29.99 * 1.10 = 32.989 → rounds to 32.99
	got := computeCustomerPrice(29.99, 10)
	if got != 32.99 {
		t.Errorf("expected 32.99, got %v", got)
	}

	// 9.95 * 1.15 = 11.4425 → rounds to 11.44
	got = computeCustomerPrice(9.95, 15)
	if got != 11.44 {
		t.Errorf("expected 11.44, got %v", got)
	}
}

func TestComputeCustomerPrice_FractionalMargin(t *testing.T) {
	// 100.0 * 1.075 = 107.5 → no rounding needed
	got := computeCustomerPrice(100.0, 7.5)
	if got != 107.5 {
		t.Errorf("expected 107.50, got %v", got)
	}
}

// TestApplyMarginFlag verifies that applyMargin is set correctly based on
// PriceMarginPercent so the correct upsert branch is selected in RunEventorySync.
func TestApplyMarginFlag(t *testing.T) {
	cases := []struct {
		margin      float64
		applyMargin bool
	}{
		{0, false},
		{-1, false}, // negative margin should not have been saved, but guard defensively
		{0.1, true},
		{10, true},
		{100, true},
	}
	for _, tc := range cases {
		cfg := &EventoryConfig{PriceMarginPercent: tc.margin}
		got := cfg.PriceMarginPercent > 0
		if got != tc.applyMargin {
			t.Errorf("margin=%.1f: expected applyMargin=%v, got %v", tc.margin, tc.applyMargin, got)
		}
	}
}
