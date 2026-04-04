package services

import (
	"fmt"
	"log"
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
	mu      sync.Mutex
	stopCh  chan struct{}
	running int32 // atomic flag: 1 when a sync is in progress
	// syncFn is called by the scheduler when a sync is due. Injected so it can
	// be overridden in tests.
	syncFn func()
}

var (
	globalSchedulerOnce sync.Once
	globalScheduler     *EventoryScheduler
)

// GetEventoryScheduler returns the singleton scheduler instance.
func GetEventoryScheduler() *EventoryScheduler {
	globalSchedulerOnce.Do(func() {
		globalScheduler = &EventoryScheduler{
			syncFn: defaultEventorySync,
		}
	})
	return globalScheduler
}

// Reset stops any running ticker, reads the current sync interval from settings,
// and starts a new ticker if SyncIntervalMinutes > 0. Safe to call from any goroutine.
func (s *EventoryScheduler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop any existing ticker goroutine
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}

	cfg, err := GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Scheduler: failed to read config: %v", err)
		return
	}
	if cfg.SyncIntervalMinutes <= 0 {
		log.Printf("[EVENTORY] Scheduler: automatic sync disabled")
		return
	}

	interval := time.Duration(cfg.SyncIntervalMinutes) * time.Minute
	log.Printf("[EVENTORY] Scheduler: starting automatic sync every %v", interval)

	stopCh := make(chan struct{})
	s.stopCh = stopCh

	go func() {
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
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[EVENTORY] Scheduler: sync panicked: %v", r)
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

// Stop terminates the running scheduler goroutine if active.
func (s *EventoryScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

// TryAcquireSync attempts to set the in-progress flag (CAS 0→1).
// Returns true if the caller now owns the flag and must call ReleaseSync when done.
// Returns false if a sync is already running.
func (s *EventoryScheduler) TryAcquireSync() bool {
	return atomic.CompareAndSwapInt32(&s.running, 0, 1)
}

// ReleaseSync clears the in-progress flag. Must be called after TryAcquireSync returned true.
func (s *EventoryScheduler) ReleaseSync() {
	atomic.StoreInt32(&s.running, 0)
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

	for _, p := range products {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			skipped++
			continue
		}

		category := strings.TrimSpace(p.Category)
		description := strings.TrimSpace(p.Description)
		now := time.Now()

		var inserted bool
		upsertErr := tx.QueryRow(`
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
		`, name, supplierName, p.Price, nullableStrS(category), nullableStrS(description), now).Scan(&inserted)

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

// nullableStrS returns nil for empty strings (for nullable DB columns).
func nullableStrS(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
