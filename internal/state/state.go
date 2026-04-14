package state

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// EpochState survives restarts — persisted to STATE_FILE.
type EpochState struct {
	ChainEpoch                int64   `json:"chain_epoch"`
	EpochStartTime            float64 `json:"epoch_start_time"`
	WalletBalanceAtEpochStart float64 `json:"wallet_balance_at_epoch_start"`
	PocStartBlockHeight       int64   `json:"poc_start_block_height"`
	EpochLength               int64   `json:"epoch_length"`

	// runtime-only (not persisted)
	Valid bool `json:"-"`
}

// EpochMaxValues tracks the running maximum per epoch to produce robust snapshots.
type EpochMaxValues struct {
	InferenceCount        int64
	MissedRequests        int64
	EarnedCoins           int64
	ValidatedInferences   int64
	InvalidatedInferences int64
	CoinBalance           int64
	EpochsCompleted       int64
}

// EpochNodeState tracks node-level data for the current epoch.
type EpochNodeState struct {
	PocWeights       map[string]int64 // node_id → max weight seen
	TimeslotAssigned int
}

// NewEpochNodeState returns an initialised EpochNodeState.
func NewEpochNodeState() EpochNodeState {
	return EpochNodeState{PocWeights: make(map[string]int64)}
}

// EpochSnapshot is one completed epoch written to history file.
type EpochSnapshot struct {
	Participant           string           `json:"participant"`
	InferenceCount        int64            `json:"inference_count"`
	MissedRequests        int64            `json:"missed_requests"`
	EarnedCoins           int64            `json:"earned_coins"`
	ValidatedInferences   int64            `json:"validated_inferences"`
	InvalidatedInferences int64            `json:"invalidated_inferences"`
	CoinBalance           int64            `json:"coin_balance"`
	EpochsCompleted       int64            `json:"epochs_completed"`
	MissRatePercent       float64          `json:"miss_rate_percent"`
	TimeslotAssigned      int              `json:"timeslot_assigned"`
	PocWeights            map[string]int64 `json:"poc_weights"`
	StartTime             float64          `json:"start_time"`
	EndTime               float64          `json:"end_time"`
	DurationSeconds       float64          `json:"duration_seconds"`
	EarnedGNK             *float64         `json:"earned_gonka,omitempty"`
	RewardedGNK           *float64         `json:"rewarded_gonka,omitempty"`
	EstimatedGNK          *float64         `json:"estimated_gonka,omitempty"`
	Claimed               *int             `json:"claimed,omitempty"`
}

// History maps participant address → epoch string → snapshot.
// This allows a single exporter to retain epoch history across participant
// address changes (e.g. when a node operator receives a new validator address).
type History = map[string]map[string]*EpochSnapshot

func LoadState(path string) EpochState {
	data, err := os.ReadFile(path)
	if err != nil {
		return EpochState{}
	}
	var s EpochState
	if err := json.Unmarshal(data, &s); err != nil {
		slog.Warn("could not parse state file", "path", path, "err", err)
		return EpochState{}
	}
	s.Valid = true
	slog.Info("restored epoch state", "epoch", s.ChainEpoch, "path", path)
	return s
}

func SaveState(path string, s EpochState) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Warn("mkdir for state file", "err", err)
		return
	}
	data, err := json.Marshal(s)
	if err != nil {
		slog.Warn("marshal state", "err", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("write state file", "err", err)
	}
}

// LoadHistory reads the history file and returns a History map.
// It transparently migrates the old flat format (map[epoch]*EpochSnapshot)
// to the new nested format (map[participant]map[epoch]*EpochSnapshot).
func LoadHistory(path string) History {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(History)
	}

	// Try new nested format first.
	// New format keys are participant addresses (non-numeric strings).
	var h History
	if err := json.Unmarshal(data, &h); err == nil && len(h) > 0 {
		isNew := true
		for k := range h {
			if _, numErr := strconv.Atoi(k); numErr == nil {
				isNew = false
				break
			}
		}
		if isNew {
			total := 0
			for _, epochs := range h {
				total += len(epochs)
			}
			slog.Info("loaded epoch history", "participants", len(h), "total_epochs", total, "path", path)
			return h
		}
	}

	// Old flat format: map[epoch]*EpochSnapshot — migrate by grouping on snap.Participant.
	var oldH map[string]*EpochSnapshot
	if err := json.Unmarshal(data, &oldH); err != nil {
		slog.Warn("could not parse epoch history", "path", path, "err", err)
		return make(History)
	}
	h = make(History)
	for epochStr, snap := range oldH {
		p := snap.Participant
		if p == "" {
			continue
		}
		if h[p] == nil {
			h[p] = make(map[string]*EpochSnapshot)
		}
		h[p][epochStr] = snap
	}
	slog.Info("migrated epoch history from old format", "participants", len(h), "path", path)
	return h
}

// SaveHistory persists the history map, pruning each participant's epochs
// independently to maxEntries oldest entries.
func SaveHistory(path string, h History, maxEntries int) {
	for _, epochs := range h {
		if len(epochs) <= maxEntries {
			continue
		}
		keys := make([]int, 0, len(epochs))
		for k := range epochs {
			if n, err := strconv.Atoi(k); err == nil {
				keys = append(keys, n)
			}
		}
		sort.Ints(keys)
		for _, k := range keys[:len(keys)-maxEntries] {
			delete(epochs, fmt.Sprintf("%d", k))
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Warn("mkdir for history file", "err", err)
		return
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		slog.Warn("marshal history", "err", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("write history file", "err", err)
	}
}

// MissRate calculates the miss rate percentage.
func MissRate(inferences, missed int64) float64 {
	total := inferences + missed
	if total == 0 {
		return 0
	}
	return math.Round(float64(missed)/float64(total)*10000) / 100
}
