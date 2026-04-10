package services

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// upsertJobDeviceSQL correctness
// ---------------------------------------------------------------------------

// TestUpsertJobDeviceSQL_HasOnConflict verifies that the outtake upsert SQL
// contains an ON CONFLICT … DO UPDATE clause so that re-scanning a device
// after an intake does not fail with a unique-constraint violation.
func TestUpsertJobDeviceSQL_HasOnConflict(t *testing.T) {
	if !strings.Contains(upsertJobDeviceSQL, "ON CONFLICT") {
		t.Error("upsertJobDeviceSQL must contain an ON CONFLICT clause")
	}
	if !strings.Contains(upsertJobDeviceSQL, "DO UPDATE") {
		t.Error("upsertJobDeviceSQL must contain a DO UPDATE clause")
	}
}

// TestUpsertJobDeviceSQL_ConflictOnJobIDAndDeviceID verifies that the conflict
// target in the ON CONFLICT clause covers both jobID and deviceID (matching the
// unique constraint enforced by migration 039), rather than just checking that
// those column names appear somewhere in the SQL.
func TestUpsertJobDeviceSQL_ConflictOnJobIDAndDeviceID(t *testing.T) {
	lower := strings.ToLower(upsertJobDeviceSQL)

	conflictIdx := strings.Index(lower, "on conflict")
	if conflictIdx == -1 {
		t.Fatal("upsertJobDeviceSQL must contain an ON CONFLICT clause")
	}

	// Extract the ON CONFLICT clause up to the closing parenthesis of its
	// column list so we only inspect the target, not the rest of the SQL.
	conflictClause := lower[conflictIdx:]
	openParen := strings.Index(conflictClause, "(")
	closeParen := strings.Index(conflictClause, ")")
	if openParen == -1 || closeParen == -1 || closeParen < openParen {
		t.Fatal("ON CONFLICT clause must have a parenthesised column target")
	}
	target := conflictClause[openParen+1 : closeParen]

	if !strings.Contains(target, "jobid") {
		t.Errorf("ON CONFLICT target %q must include jobid", target)
	}
	if !strings.Contains(target, "deviceid") {
		t.Errorf("ON CONFLICT target %q must include deviceid", target)
	}
}

// TestUpsertJobDeviceSQL_UpdatesPackStatusToIssued verifies that the DO UPDATE
// clause resets pack_status to 'issued' so that a device returned via intake
// (which sets pack_status = 'pending') is correctly marked as issued again.
func TestUpsertJobDeviceSQL_UpdatesPackStatusToIssued(t *testing.T) {
	if !strings.Contains(upsertJobDeviceSQL, "pack_status = 'issued'") {
		t.Error("DO UPDATE must set pack_status = 'issued'")
	}
}

// TestUpsertJobDeviceSQL_UpdatesPackTs verifies that the DO UPDATE clause
// explicitly sets pack_ts (e.g. to NOW()) so the timestamp always reflects the
// actual scan time, not just that the column name appears in the INSERT list.
func TestUpsertJobDeviceSQL_UpdatesPackTs(t *testing.T) {
	lower := strings.ToLower(upsertJobDeviceSQL)

	doUpdateIdx := strings.Index(lower, "do update")
	if doUpdateIdx == -1 {
		t.Fatal("upsertJobDeviceSQL must contain a DO UPDATE clause")
	}

	doUpdateClause := lower[doUpdateIdx:]
	if !strings.Contains(doUpdateClause, "set") {
		t.Error("DO UPDATE clause must contain a SET")
	}
	if !strings.Contains(doUpdateClause, "pack_ts") {
		t.Error("DO UPDATE SET must include pack_ts")
	}
	if !strings.Contains(doUpdateClause, "now()") {
		t.Error("DO UPDATE SET must assign pack_ts = NOW()")
	}
}

// TestUpsertJobDeviceSQL_TargetsJobdevicesTable verifies that the INSERT is
// directed at the `jobdevices` table (not `job_devices` or any other alias).
func TestUpsertJobDeviceSQL_TargetsJobdevicesTable(t *testing.T) {
	lower := strings.ToLower(upsertJobDeviceSQL)
	if !strings.Contains(lower, "jobdevices") {
		t.Error("upsertJobDeviceSQL must target the jobdevices table")
	}
	if strings.Contains(lower, "job_devices") {
		t.Error("upsertJobDeviceSQL must not reference job_devices (wrong table name)")
	}
}

// ---------------------------------------------------------------------------
// processOuttake validation
// ---------------------------------------------------------------------------

// TestProcessOuttake_NilJobIDReturnsError verifies that processOuttake returns
// a descriptive error when no job ID is provided, so callers receive a clear
// failure response instead of a nil-pointer panic.
func TestProcessOuttake_NilJobIDReturnsError(t *testing.T) {
	svc := &ScanService{} // db is nil; the nil-jobID guard fires before any DB call
	_, _, err := svc.processOuttake(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when jobID is nil, got nil")
	}
	if !strings.Contains(err.Error(), "job_id is required") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}
