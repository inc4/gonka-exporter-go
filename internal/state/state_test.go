package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- MissRate ---

func TestMissRate_ZeroTotal(t *testing.T) {
	if got := MissRate(0, 0); got != 0 {
		t.Fatalf("MissRate(0,0) = %v, want 0", got)
	}
}

func TestMissRate_NoMisses(t *testing.T) {
	if got := MissRate(100, 0); got != 0 {
		t.Fatalf("MissRate(100,0) = %v, want 0", got)
	}
}

func TestMissRate_AllMissed(t *testing.T) {
	if got := MissRate(0, 50); got != 100 {
		t.Fatalf("MissRate(0,50) = %v, want 100", got)
	}
}

func TestMissRate_Half(t *testing.T) {
	if got := MissRate(50, 50); got != 50 {
		t.Fatalf("MissRate(50,50) = %v, want 50", got)
	}
}

func TestMissRate_Rounded(t *testing.T) {
	// 1/3 ≈ 33.33%
	got := MissRate(2, 1)
	if got != 33.33 {
		t.Fatalf("MissRate(2,1) = %v, want 33.33", got)
	}
}

// --- LoadHistory / SaveHistory ---

func TestLoadHistory_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	h := LoadHistory(filepath.Join(dir, "nonexistent.json"))
	if h == nil {
		t.Fatal("expected non-nil History map")
	}
	if len(h) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(h))
	}
}

func TestLoadHistory_NewFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := History{
		"addr1": {
			"10": &EpochSnapshot{Participant: "addr1", InferenceCount: 100},
			"11": &EpochSnapshot{Participant: "addr1", InferenceCount: 200},
		},
	}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	loaded := LoadHistory(path)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(loaded))
	}
	if len(loaded["addr1"]) != 2 {
		t.Fatalf("expected 2 epochs, got %d", len(loaded["addr1"]))
	}
	if loaded["addr1"]["10"].InferenceCount != 100 {
		t.Fatalf("wrong inference count: %d", loaded["addr1"]["10"].InferenceCount)
	}
}

func TestLoadHistory_OldFlatFormat_Migration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Old format: map[epoch_string]*EpochSnapshot (keys are numeric strings)
	old := map[string]*EpochSnapshot{
		"5": {Participant: "addr_old", InferenceCount: 42},
		"6": {Participant: "addr_old", InferenceCount: 55},
	}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	loaded := LoadHistory(path)
	if len(loaded) != 1 {
		t.Fatalf("migration: expected 1 participant, got %d", len(loaded))
	}
	epochs := loaded["addr_old"]
	if len(epochs) != 2 {
		t.Fatalf("migration: expected 2 epochs, got %d", len(epochs))
	}
	if epochs["5"].InferenceCount != 42 {
		t.Fatalf("migration: wrong inference count for epoch 5: %d", epochs["5"].InferenceCount)
	}
}

func TestSaveHistory_Pruning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Build 10 epochs for one participant, max=3
	h := History{"addr1": make(map[string]*EpochSnapshot)}
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("%d", i)
		h["addr1"][key] = &EpochSnapshot{Participant: "addr1", InferenceCount: int64(i)}
	}

	SaveHistory(path, h, 3)

	loaded := LoadHistory(path)
	if len(loaded["addr1"]) != 3 {
		t.Fatalf("pruning: expected 3 epochs after maxEntries=3, got %d", len(loaded["addr1"]))
	}
	// Must keep the 3 highest epoch numbers (8, 9, 10)
	for _, epoch := range []string{"8", "9", "10"} {
		if loaded["addr1"][epoch] == nil {
			t.Fatalf("pruning: expected epoch %s to be retained", epoch)
		}
	}
}

func TestSaveHistory_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	snap := &EpochSnapshot{
		Participant:    "addr_test",
		InferenceCount: 999,
		MissRatePercent: 12.5,
	}
	h := History{"addr_test": {"42": snap}}

	SaveHistory(path, h, 500)
	loaded := LoadHistory(path)

	got := loaded["addr_test"]["42"]
	if got == nil {
		t.Fatal("roundtrip: epoch 42 not found")
	}
	if got.InferenceCount != 999 {
		t.Fatalf("roundtrip: InferenceCount = %d, want 999", got.InferenceCount)
	}
	if got.MissRatePercent != 12.5 {
		t.Fatalf("roundtrip: MissRatePercent = %v, want 12.5", got.MissRatePercent)
	}
}
