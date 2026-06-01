package yield

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeSnap(dateStr string, value float64) Snapshot {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	return Snapshot{Date: t.UTC(), TotalValue: value}
}

// yearStart builds the Jan-1 00:00:00 boundary in the given location,
// mirroring how yield.go constructs periodStart.
func yearStart(year int, loc *time.Location) time.Time {
	return time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
}

func TestLoad_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	snaps, err := store.load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(snaps))
	}
}

func TestAppend_ThenLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	s1 := makeSnap("2025-01-01", 100000.0)
	s2 := makeSnap("2025-06-01", 120000.0)
	s3 := makeSnap("2025-12-31", 130000.0)

	for _, s := range []Snapshot{s1, s2, s3} {
		if err := store.append(s); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	snaps, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}
	if snaps[0].TotalValue != 100000.0 {
		t.Errorf("snaps[0].TotalValue = %v, want 100000", snaps[0].TotalValue)
	}
	if snaps[1].TotalValue != 120000.0 {
		t.Errorf("snaps[1].TotalValue = %v, want 120000", snaps[1].TotalValue)
	}
	if snaps[2].TotalValue != 130000.0 {
		t.Errorf("snaps[2].TotalValue = %v, want 130000", snaps[2].TotalValue)
	}
}

func TestAppend_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	// Use a nested path that doesn't exist yet.
	store := newSnapshotStore(filepath.Join(dir, "sub", "nested", "snapshots.json"))

	if err := store.append(makeSnap("2025-01-01", 50000.0)); err != nil {
		t.Fatalf("append to nested path failed: %v", err)
	}
	snaps, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
}

func TestAppend_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	if err := store.append(makeSnap("2025-01-01", 100000.0)); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	// The temp file must NOT remain after the atomic rename.
	tmpPath := store.path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s should not exist after append", tmpPath)
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshots.json")
	if err := os.WriteFile(path, []byte("{not valid json["), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newSnapshotStore(path)

	_, err := store.load()
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestValueAtYearStart_LastOnOrBefore(t *testing.T) {
	// Dec 30 prior year and Jan 5 current year -> should pick Dec 30.
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2024-12-30", 111111.0))
	_ = store.append(makeSnap("2025-01-05", 222222.0))

	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value != 111111.0 {
		t.Errorf("value = %v, want 111111", value)
	}
}

func TestValueAtYearStart_EarlyJanuaryFallback(t *testing.T) {
	// Only a snapshot within first 7 days of Jan, no prior-year snapshot.
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2025-01-03", 99999.0))

	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value != 99999.0 {
		t.Errorf("value = %v, want 99999", value)
	}
}

func TestValueAtYearStart_EarlyJanuaryFallback_OutsideTolerance(t *testing.T) {
	// Only a snapshot on Jan 10 — beyond 7-day tolerance — should not match.
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2025-01-10", 99999.0))

	_, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for snapshot outside tolerance")
	}
}

func TestValueAtYearStart_ManualOverride(t *testing.T) {
	// No qualifying snapshot, manualOverride > 0 -> return override.
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2024-01-01", 50000.0)) // too old for 2025 start

	// No Dec 2024 or early Jan 2025 snapshot.
	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 77777.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true with manualOverride")
	}
	if value != 77777.0 {
		t.Errorf("value = %v, want 77777", value)
	}
}

func TestValueAtYearStart_NoDataNoOverride(t *testing.T) {
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false with no data and no override")
	}
	if value != 0 {
		t.Errorf("value = %v, want 0", value)
	}
}

func TestValueAtYearStart_OutOfOrderSnapshots(t *testing.T) {
	// Write snapshots out of chronological order; resolution must still be correct.
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshots.json")

	// Write directly to disk in reversed order.
	snaps := []Snapshot{
		makeSnap("2025-01-05", 300000.0),
		makeSnap("2024-12-31", 200000.0), // should be picked: last on/before Jan 1 2025
		makeSnap("2024-11-15", 150000.0),
	}
	data, _ := json.MarshalIndent(snaps, "", "  ")
	_ = os.WriteFile(path, data, 0o644)

	store := newSnapshotStore(path)
	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value != 200000.0 {
		t.Errorf("value = %v, want 200000 (Dec 31 snapshot)", value)
	}
}

func TestValueAtYearStart_MultipleBeforeTarget_PicksLast(t *testing.T) {
	// Several snapshots before target; must pick the latest one (Dec 31, not Dec 15).
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2024-11-01", 100000.0))
	_ = store.append(makeSnap("2024-12-15", 110000.0))
	_ = store.append(makeSnap("2024-12-31", 120000.0))
	_ = store.append(makeSnap("2025-01-10", 130000.0))

	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value != 120000.0 {
		t.Errorf("value = %v, want 120000 (Dec 31 snapshot)", value)
	}
}

func TestValueAtYearStart_MultipleEarlyJanuary_PicksEarliest(t *testing.T) {
	// No prior-year snapshot; two early-Jan snapshots — must pick the earliest.
	dir := t.TempDir()
	store := newSnapshotStore(filepath.Join(dir, "snapshots.json"))

	_ = store.append(makeSnap("2025-01-05", 222222.0))
	_ = store.append(makeSnap("2025-01-02", 111111.0)) // earliest

	value, ok, err := store.valueAtYearStart(yearStart(2025, time.UTC), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value != 111111.0 {
		t.Errorf("value = %v, want 111111 (Jan 2 snapshot)", value)
	}
}

// TestValueAtYearStart_NonUTCTimezone_PositiveOffset is a regression test for the
// timezone mismatch bug. When the system is in a positive UTC offset (e.g. MSK =
// UTC+3), a snapshot taken at Jan 1 02:00 local time is Dec 31 23:00 UTC. With
// the old code that built target in time.UTC, that snapshot would be treated as
// "on or before Jan 1 UTC" and incorrectly selected as the prior-year closing
// value. The fix passes yearStart in the caller's location, so the comparison is
// timezone-consistent and the Dec 31 snapshot is correctly picked instead.
func TestValueAtYearStart_NonUTCTimezone_PositiveOffset(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600) // UTC+3

	dir := t.TempDir()
	path := filepath.Join(dir, "snapshots.json")

	// Dec 31 2024 22:00 MSK = Dec 31 2024 19:00 UTC — unambiguously prior year.
	dec31 := time.Date(2024, time.December, 31, 22, 0, 0, 0, loc)
	// Jan 1 2025 02:00 MSK = Dec 31 2024 23:00 UTC. The bug would pick this as
	// "on or before Jan 1 UTC" — selecting the wrong (current-year) snapshot.
	jan1Morning := time.Date(2025, time.January, 1, 2, 0, 0, 0, loc)

	snaps := []Snapshot{
		{Date: dec31, TotalValue: 100000.0},
		{Date: jan1Morning, TotalValue: 200000.0},
	}
	data, _ := json.MarshalIndent(snaps, "", "  ")
	_ = os.WriteFile(path, data, 0o644)

	store := newSnapshotStore(path)

	// yearStart in MSK: Jan 1 2025 00:00 MSK.
	ys := time.Date(2025, time.January, 1, 0, 0, 0, 0, loc)
	value, ok, err := store.valueAtYearStart(ys, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Must return the Dec 31 22:00 MSK snapshot (100000), NOT the Jan 1 02:00
	// MSK one (200000). The Jan 1 02:00 MSK snapshot is after midnight local
	// time and falls into the early-January fallback window, but since there IS
	// a valid prior-year snapshot it should never be selected as yearStart value.
	if value != 100000.0 {
		t.Errorf("value = %v, want 100000 (Dec 31 MSK snapshot); got current-year snapshot instead — timezone bug not fixed", value)
	}
}
