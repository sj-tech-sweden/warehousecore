package services

import (
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"warehousecore/internal/repository"
)

// EventoryScheduler runs a background goroutine that periodically syncs
// products from the Eventory API. It is a singleton; call GetEventoryScheduler()
// to obtain the shared instance.
type EventoryScheduler struct {
	mu       sync.Mutex
	resetMu  sync.Mutex // serializes concurrent Reset() calls to prevent tickerWg Add/Wait races
	stopCh   chan struct{}
	stopped  bool           // set by Stop(); prevents new Add() calls on tickerWg / wg
	tickerWg sync.WaitGroup // tracks the ticker goroutine itself
	wg       sync.WaitGroup // tracks goroutines running syncFn so Stop() can wait
	running  int32          // atomic flag: 1 when a sync is in progress
	// syncFn is called by the scheduler when a sync is due. Injected so it can
	// be overridden in tests.
	syncFn func()
	// configFn reads the Eventory config. Defaults to GetEventoryConfig; can be
	// overridden in tests to avoid a real database connection.
	configFn func() (*EventoryConfig, error)
}

var (
	globalSchedulerOnce sync.Once
	globalScheduler     *EventoryScheduler
)

// allowedSyncIntervals is the set of valid sync_interval_minutes values that
// the scheduler accepts. Validating against this set prevents time.Duration
// overflow and time.NewTicker panicking on a zero/negative duration from an
// out-of-range value stored in the database.
// Matches the options exposed in the admin UI: 15 min / 30 min / 1 h / 2 h /
// 4 h / 8 h / 24 h.
var allowedSyncIntervals = map[int]bool{
	15:   true,
	30:   true,
	60:   true,
	120:  true,
	240:  true,
	480:  true,
	1440: true,
}

// IsAllowedSyncInterval returns true when n is a valid non-zero sync interval.
// Exposed so the HTTP handler can validate user-supplied values before saving.
func IsAllowedSyncInterval(n int) bool {
	return allowedSyncIntervals[n]
}

// GetEventoryScheduler returns the singleton scheduler instance.
func GetEventoryScheduler() *EventoryScheduler {
	globalSchedulerOnce.Do(func() {
		globalScheduler = &EventoryScheduler{
			syncFn:   defaultEventorySync,
			configFn: GetEventoryConfig,
		}
	})
	return globalScheduler
}

// Reset stops any running ticker, reads the current sync interval from settings,
// and starts a new ticker if SyncIntervalMinutes > 0. Safe to call from any goroutine.
// Reset is a no-op once Stop() has been called.
//
// resetMu serializes concurrent Reset() calls. This prevents the tickerWg
// Add/Wait race: without serialisation, one goroutine could be returning from
// tickerWg.Wait() (counter just hit zero) while another concurrently calls
// tickerWg.Add(1), which the Go docs flag as "WaitGroup misuse" and can panic.
// With resetMu held for the entire Reset, a second Reset() cannot call Add(1)
// until the first Reset()'s Wait() has fully returned.
//
// After signalling the old ticker goroutine to stop, Reset releases s.mu and
// waits for the goroutine to fully exit before starting a new one. This makes
// the reset deterministic: Go's select is pseudo-random when both ticker.C and
// stopCh are simultaneously ready, so without this wait the old goroutine could
// still fire one extra sync after Reset returns.
func (s *EventoryScheduler) Reset() {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()

	s.mu.Lock()

	// Do not start a new ticker if the scheduler has been stopped.
	if s.stopped {
		s.mu.Unlock()
		return
	}

	// Signal any existing ticker goroutine to stop.
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}

	s.mu.Unlock()

	// Wait for the old ticker goroutine to fully exit before starting a new one.
	// This is safe to call without the mutex: tickerWg.Add(1) only ever happens
	// inside Reset() under s.mu, so no new Add can race with this Wait.
	s.tickerWg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Re-check stopped in case Stop() was called while we were waiting.
	if s.stopped {
		return
	}

	cfg, err := s.configFn()
	if err != nil {
		log.Printf("[EVENTORY] Scheduler: failed to read config: %v", err)
		return
	}
	if cfg.SyncIntervalMinutes <= 0 {
		log.Printf("[EVENTORY] Scheduler: automatic sync disabled")
		return
	}

	// Validate against the known supported set to prevent time.Duration overflow
	// or time.NewTicker panicking on a zero/negative duration from an invalid value.
	if !allowedSyncIntervals[cfg.SyncIntervalMinutes] {
		log.Printf("[EVENTORY] Scheduler: unsupported sync_interval_minutes=%d, disabling automatic sync", cfg.SyncIntervalMinutes)
		return
	}

	interval := time.Duration(cfg.SyncIntervalMinutes) * time.Minute
	log.Printf("[EVENTORY] Scheduler: starting automatic sync every %v", interval)

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
				// Skip tick if a sync is already in progress to avoid overlap.
				if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
					log.Printf("[EVENTORY] Scheduler: skipping tick, sync already in progress")
					continue
				}
				log.Printf("[EVENTORY] Scheduler: running scheduled sync")
				// Add to WaitGroup before launching the goroutine so that
				// Stop()/wg.Wait() cannot return before the sync starts.
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[EVENTORY] Scheduler: sync panicked: %v\n%s", r, debug.Stack())
						}
						atomic.StoreInt32(&s.running, 0)
					}()
					s.syncFn()
				}()
			case <-stopCh:
				log.Printf("[EVENTORY] Scheduler: stopped")
				return
			}
		}
	}()
}

// Stop terminates the running ticker goroutine if active, then waits for
// any in-progress sync to finish before returning. This ensures that an
// in-flight RunEventorySync call completes (and its DB transaction is
// committed or rolled back) before the caller proceeds with teardown such as
// closing the database connection.
//
// The two-phase wait is safe because:
//  1. stopped is set under s.mu before the mutex is released, so neither
//     Reset() (tickerWg.Add) nor TryAcquireSync() (wg.Add) can call Add
//     after Stop begins — both check stopped under s.mu and return early.
//  2. tickerWg.Wait() blocks until the ticker goroutine exits, guaranteeing
//     that no further wg.Add(1) calls will be made from the scheduled-sync path.
//  3. wg.Wait() then blocks until any sync goroutines that were already
//     running (and whose wg.Add(1) preceded Stop()) complete.
//
// Manual syncs via TryAcquireSync only occur during HTTP request handling,
// which completes before graceful shutdown calls Stop().
func (s *EventoryScheduler) Stop() {
	s.mu.Lock()
	s.stopped = true
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.mu.Unlock()
	// Phase 1: wait for the ticker goroutine to exit so it can no longer call
	// s.wg.Add(1). This prevents the Add/Wait race described in the Go docs.
	s.tickerWg.Wait()
	// Phase 2: wait for any sync goroutines that were already launched.
	s.wg.Wait()
}

// TryAcquireSync attempts to set the in-progress flag (CAS 0→1) and increments
// the WaitGroup so that Stop() will wait for this manual sync too.
// Returns true if the caller now owns the flag and must call ReleaseSync when done.
// Returns false if a sync is already running or the scheduler has been stopped
// (WaitGroup is not incremented in either case).
func (s *EventoryScheduler) TryAcquireSync() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return false
	}
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return false
	}
	s.wg.Add(1)
	return true
}

// ReleaseSync clears the in-progress flag and signals the WaitGroup.
// Must be called after TryAcquireSync returned true.
func (s *EventoryScheduler) ReleaseSync() {
	atomic.StoreInt32(&s.running, 0)
	s.wg.Done()
}

// defaultEventorySync performs the actual product sync and logs the result.
func defaultEventorySync() {
	cfg, err := GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Scheduled sync: failed to load config: %v", err)
		return
	}
	if cfg.APIURL == "" {
		log.Printf("[EVENTORY] Scheduled sync: API URL not configured, skipping")
		return
	}

	imported, updated, skipped, total, err := RunEventorySync(cfg)
	if err != nil {
		log.Printf("[EVENTORY] Scheduled sync failed: %v", err)
		return
	}
	log.Printf("[EVENTORY] Scheduled sync complete: %d/%d imported, %d updated, %d skipped",
		imported, total, updated, skipped)
}

// applyMarginPrice computes the customer price from a base rental price and a
// margin percentage, rounding to two decimal places. It uses decimal-safe
// rounding (format then re-parse) to avoid binary float representation errors.
//
//	applyMarginPrice(100.0, 10) → 110.00
//	applyMarginPrice(29.99, 10) → 32.99
func applyMarginPrice(rentalPrice, marginPercent float64) float64 {
	result := rentalPrice * (1 + marginPercent/100)
	rounded, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", result), 64)
	return rounded
}

// RunEventorySync fetches products from Eventory and upserts them into the
// rental_equipment table. It returns counts of imported, updated, and skipped
// rows plus any fatal error. This is extracted from the handler so it can be
// reused by the scheduler.
func RunEventorySync(cfg *EventoryConfig) (imported, updated, skipped, total int, err error) {
	products, err := FetchEventoryProducts(cfg)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	total = len(products)

	db := repository.GetSQLDB()
	if db == nil {
		return 0, 0, 0, total, ErrDatabaseNotAvailable
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, 0, total, fmt.Errorf("failed to begin transaction: %w", err)
	}

	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("[EVENTORY] Sync rollback failed: %v", rbErr)
		}
	}

	supplierName := cfg.EffectiveSupplierName()
	applyMargin := cfg.PriceMarginPercent > 0
	now := time.Now()

	// Build the upsert query once outside the loop. When a price margin is
	// configured, customer_price is computed from rental_price and also updated
	// on conflict. When margin is 0, customer_price is left at 0 on insert and
	// not touched on subsequent syncs so manually-set prices are preserved.
	var upsertQuery string
	if applyMargin {
		upsertQuery = `
			INSERT INTO rental_equipment
				(product_name, supplier_name, rental_price, customer_price, category, description, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, $7)
			ON CONFLICT (product_name, supplier_name)
			DO UPDATE SET
				rental_price   = EXCLUDED.rental_price,
				customer_price = EXCLUDED.customer_price,
				category       = EXCLUDED.category,
				description    = EXCLUDED.description,
				updated_at     = EXCLUDED.updated_at
			RETURNING (xmax = 0) AS inserted
		`
	} else {
		upsertQuery = `
			INSERT INTO rental_equipment
				(product_name, supplier_name, rental_price, customer_price, category, description, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, 0, $4, $5, TRUE, $6, $6)
			ON CONFLICT (product_name, supplier_name)
			DO UPDATE SET
				rental_price = EXCLUDED.rental_price,
				category     = EXCLUDED.category,
				description  = EXCLUDED.description,
				updated_at   = EXCLUDED.updated_at
			RETURNING (xmax = 0) AS inserted
		`
	}

	for _, p := range products {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			skipped++
			continue
		}

		category := strings.TrimSpace(p.Category)
		description := strings.TrimSpace(p.Description)

		var inserted bool
		var upsertErr error
		if applyMargin {
			customerPrice := applyMarginPrice(p.Price, cfg.PriceMarginPercent)
			upsertErr = tx.QueryRow(upsertQuery, name, supplierName, p.Price, customerPrice,
				nullableString(&category), nullableString(&description), now).Scan(&inserted)
		} else {
			upsertErr = tx.QueryRow(upsertQuery, name, supplierName, p.Price,
				nullableString(&category), nullableString(&description), now).Scan(&inserted)
		}

		if upsertErr != nil {
			log.Printf("[EVENTORY] Failed to upsert product %q: %v", name, upsertErr)
			rollback()
			return imported, updated, skipped, total, fmt.Errorf("upsert failed for %q: %w", name, upsertErr)
		}

		if inserted {
			imported++
		} else {
			updated++
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		rollback()
		return 0, 0, 0, total, fmt.Errorf("failed to commit sync transaction: %w", commitErr)
	}

	return imported, updated, skipped, total, nil
}
