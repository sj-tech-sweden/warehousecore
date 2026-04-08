package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
)

// deviceIDLikeEscaper escapes SQL LIKE wildcard characters (\, %, _) so that
// a device ID prefix derived from user or DB input is treated as a literal
// string in a LIKE predicate.
var deviceIDLikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// numericSuffixPattern is the PostgreSQL regular-expression used to test
// whether the suffix after the device ID prefix is a pure decimal integer.
const numericSuffixPattern = `^[0-9]+$`

// deriveDeviceIDPrefix returns the device ID prefix for a given product.
// If manualPrefix is non-empty it is returned verbatim (after trimming).
// Otherwise the prefix is derived from the product's subcategory abbreviation
// + pos_in_category (e.g. "LED1"). If no abbreviation is found the function
// falls back to "P{productID}" rather than raising an error, intentionally
// diverging from the DB trigger (migration 030) which raises in that case.
//
// The caller must hold an open transaction (tx).
func deriveDeviceIDPrefix(ctx context.Context, tx *sql.Tx, productID int, manualPrefix string) (string, error) {
	if p := strings.TrimSpace(manualPrefix); p != "" {
		return p, nil
	}

	var abbreviation sql.NullString
	var posInCategory sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT s.abbreviation, p.pos_in_category
		FROM products p
		LEFT JOIN subcategories s ON p.subcategoryID = s.subcategoryID
		WHERE p.productID = $1
	`, productID).Scan(&abbreviation, &posInCategory)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("product %d not found", productID)
		}
		return "", fmt.Errorf("failed to fetch product info: %w", err)
	}

	if abbreviation.Valid && abbreviation.String != "" {
		posStr := "0"
		if posInCategory.Valid {
			posStr = fmt.Sprintf("%d", posInCategory.Int64)
		}
		return abbreviation.String + posStr, nil
	}
	return fmt.Sprintf("P%d", productID), nil
}

// allocateDeviceCounter acquires a pg_advisory_xact_lock keyed on a FNV-64a
// hash of the prefix to serialize concurrent allocation, then returns the next
// available numeric counter for device IDs that start with prefix.
//
// The existing counter is found by scanning deviceIDs with an index-friendly
// LIKE predicate (enabling use of the btree index on deviceID). Wildcard
// characters (\, %, _) in the prefix are escaped so they are treated as
// literals. The numeric suffix after the prefix can be any length; counters
// above 999 are handled naturally.
//
// New IDs should be formatted with fmt.Sprintf("%s%03d", prefix, counter+i)
// to preserve three-digit leading zeros for counters below 1000, matching
// the DB trigger's LPAD(..., 3, '0') behaviour.
//
// The caller must hold an open transaction (tx).
func allocateDeviceCounter(ctx context.Context, tx *sql.Tx, prefix string) (int64, error) {
	// Serialize concurrent calls for the same prefix: key the advisory lock on
	// the FNV-64a hash of the prefix so different products that share the same
	// prefix namespace are also correctly serialized.  We mask the hash to the
	// non-negative int64 range so the lock key is always positive and
	// unambiguously scoped to device-ID allocation (avoiding any accidental
	// overlap with locks that might use negative keys for other purposes).
	h := fnv.New64a()
	h.Write([]byte(prefix))
	lockKey := int64(h.Sum64() & 0x7fffffffffffffff)
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey); err != nil {
		return 0, fmt.Errorf("failed to acquire device creation lock: %w", err)
	}

	// Build a LIKE pattern that treats the prefix as a literal string.
	// We escape \, %, and _ so they are not interpreted as wildcard characters
	// by PostgreSQL, then append % so the predicate matches any device ID that
	// starts with the prefix. Using LIKE allows PostgreSQL to use the btree
	// index on deviceID for a prefix scan, unlike the non-sargable
	// LEFT(deviceID, CHAR_LENGTH(prefix)) = prefix form.
	escapedPrefix := deviceIDLikeEscaper.Replace(prefix)
	pattern := escapedPrefix + "%"

	var nextCounter int64
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(
			CASE
				WHEN SUBSTRING(deviceID FROM CHAR_LENGTH($1) + 1) ~ '`+numericSuffixPattern+`'
				THEN CAST(SUBSTRING(deviceID FROM CHAR_LENGTH($1) + 1) AS BIGINT)
				ELSE 0
			END
		), 0) + 1
		FROM devices
		WHERE deviceID LIKE $2 ESCAPE '\'
	`, prefix, pattern).Scan(&nextCounter)
	if err != nil {
		return 0, fmt.Errorf("failed to determine next device counter: %w", err)
	}
	return nextCounter, nil
}
