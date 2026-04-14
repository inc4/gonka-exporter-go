package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// flexInt64 accepts both JSON number and JSON string for the same field.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*f = flexInt64(n)
	return nil
}

func get(url string, dest any) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

// --- Tendermint RPC ---

type TendermintStatus struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			LatestBlockTime   string `json:"latest_block_time"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

func FetchTendermintStatus(rpcURL string) (*TendermintStatus, error) {
	var s TendermintStatus
	err := get(rpcURL+"/status", &s)
	return &s, err
}

func FetchBlockTimeAtHeight(rpcURL string, height int64) (float64, error) {
	var resp struct {
		Result struct {
			Block struct {
				Header struct {
					Time string `json:"time"`
				} `json:"header"`
			} `json:"block"`
		} `json:"result"`
	}
	if err := get(fmt.Sprintf("%s/block?height=%d", rpcURL, height), &resp); err != nil {
		return 0, err
	}
	t := resp.Result.Block.Header.Time
	if t == "" {
		return 0, fmt.Errorf("empty time")
	}
	t = strings.TrimSuffix(t, "Z")
	parsed, err := time.Parse("2006-01-02T15:04:05.999999999", t)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, t+"Z")
		if err != nil {
			return 0, fmt.Errorf("parse time %q: %w", t, err)
		}
	}
	return float64(parsed.UTC().Unix()) + float64(parsed.Nanosecond())/1e9, nil
}

func FetchMaxBlockHeightFromNodes(nodes []string) (int64, string) {
	sample := rand.Perm(len(nodes))
	if len(sample) > 5 {
		sample = sample[:5]
	}
	var maxHeight int64
	var latestTime string
	for _, i := range sample {
		var resp struct {
			Result struct {
				SyncInfo struct {
					LatestBlockHeight string `json:"latest_block_height"`
					LatestBlockTime   string `json:"latest_block_time"`
				} `json:"sync_info"`
			} `json:"result"`
		}
		if err := get(nodes[i]+"/chain-rpc/status", &resp); err != nil {
			continue
		}
		h, err := strconv.ParseInt(resp.Result.SyncInfo.LatestBlockHeight, 10, 64)
		if err == nil && h > maxHeight {
			maxHeight = h
			latestTime = resp.Result.SyncInfo.LatestBlockTime
		}
	}
	return maxHeight, latestTime
}

// --- Chain REST ---

func FetchCurrentEpoch(restURL string) (int64, error) {
	var r struct {
		Epoch string `json:"epoch"`
	}
	if err := get(restURL+"/productscience/inference/inference/get_current_epoch", &r); err != nil {
		return 0, err
	}
	return strconv.ParseInt(r.Epoch, 10, 64)
}

// EpochInfo contains epoch block data from the chain.
type EpochInfo struct {
	PocStartBlockHeight int64
	EpochLength         int64
	BlockHeight         int64
}

func FetchEpochInfo(restURL string) (*EpochInfo, error) {
	var r struct {
		LatestEpoch struct {
			PocStartBlockHeight string `json:"poc_start_block_height"`
		} `json:"latest_epoch"`
		Params struct {
			EpochParams struct {
				EpochLength string `json:"epoch_length"`
			} `json:"epoch_params"`
		} `json:"params"`
		BlockHeight string `json:"block_height"`
	}
	if err := get(restURL+"/productscience/inference/inference/epoch_info", &r); err != nil {
		return nil, err
	}
	poc, err1 := strconv.ParseInt(r.LatestEpoch.PocStartBlockHeight, 10, 64)
	el, err2  := strconv.ParseInt(r.Params.EpochParams.EpochLength, 10, 64)
	bh, err3  := strconv.ParseInt(r.BlockHeight, 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, fmt.Errorf("parse epoch_info: %v %v %v", err1, err2, err3)
	}
	return &EpochInfo{
		PocStartBlockHeight: poc,
		EpochLength:         el,
		BlockHeight:         bh,
	}, nil
}

// --- Epoch group data ---

// ValidationWeight is a per-participant weight entry.
type ValidationWeight struct {
	MemberAddress string    `json:"member_address"`
	Weight        string    `json:"weight"`
	Reputation    flexInt64 `json:"reputation"`
}

// EpochGroupData contains network-wide epoch weight data.
type EpochGroupData struct {
	TotalWeight       int64
	EpochIndex        int64
	NumberOfRequests  int64
	ValidationWeights []ValidationWeight
}

func FetchEpochGroupData(restURL string) (*EpochGroupData, error) {
	var r struct {
		EpochGroupData struct {
			TotalWeight       string             `json:"total_weight"`
			EpochIndex        string             `json:"epoch_index"`
			NumberOfRequests  flexInt64          `json:"number_of_requests"`
			ValidationWeights []ValidationWeight `json:"validation_weights"`
		} `json:"epoch_group_data"`
	}
	if err := get(restURL+"/productscience/inference/inference/current_epoch_group_data", &r); err != nil {
		return nil, err
	}
	tw, _ := strconv.ParseInt(r.EpochGroupData.TotalWeight, 10, 64)
	ei, _ := strconv.ParseInt(r.EpochGroupData.EpochIndex, 10, 64)
	return &EpochGroupData{
		TotalWeight:       tw,
		EpochIndex:        ei,
		NumberOfRequests:  int64(r.EpochGroupData.NumberOfRequests),
		ValidationWeights: r.EpochGroupData.ValidationWeights,
	}, nil
}

// --- Epoch performance summary ---

// EpochPerfSummary holds on-chain reward data for a completed epoch.
type EpochPerfSummary struct {
	RewardedGNK float64
	Claimed     int
}

func FetchEpochPerfSummary(restURL, address string, epochNum int64) (*EpochPerfSummary, error) {
	var r struct {
		EpochPerformanceSummary struct {
			RewardedCoins string `json:"rewarded_coins"`
			Claimed       bool   `json:"claimed"`
		} `json:"epochPerformanceSummary"`
	}
	url := fmt.Sprintf("%s/productscience/inference/inference/epoch_performance_summary/%d/%s", restURL, epochNum, address)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	rc, err := strconv.ParseInt(r.EpochPerformanceSummary.RewardedCoins, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse rewarded_coins: %w", err)
	}
	claimed := 0
	if r.EpochPerformanceSummary.Claimed {
		claimed = 1
	}
	return &EpochPerfSummary{RewardedGNK: float64(rc) / 1e9, Claimed: claimed}, nil
}

// --- Participant stats ---

// ParticipantStats holds current-epoch stats for one participant.
type ParticipantStats struct {
	EpochsCompleted              int64
	CoinBalance                  int64
	InferenceCount               int64
	MissedRequests               int64
	EarnedCoins                  int64
	ValidatedInferences          int64
	InvalidatedInferences        int64
	Status                       string // "ACTIVE", "INACTIVE", "INVALID", "UNCONFIRMED", "UNSPECIFIED"
	ConsecutiveInvalidInferences int64
	BurnedCoins                  int64
	RewardedCoins                int64
}

type participantStatsResp struct {
	Participant struct {
		EpochsCompleted              flexInt64 `json:"epochs_completed"`
		CoinBalance                  flexInt64 `json:"coin_balance"`
		Status                       string    `json:"status"`
		ConsecutiveInvalidInferences flexInt64 `json:"consecutive_invalid_inferences"`
		CurrentEpochStats            struct {
			InferenceCount        flexInt64 `json:"inference_count"`
			MissedRequests        flexInt64 `json:"missed_requests"`
			EarnedCoins           flexInt64 `json:"earned_coins"`
			ValidatedInferences   flexInt64 `json:"validated_inferences"`
			InvalidatedInferences flexInt64 `json:"invalidated_inferences"`
			BurnedCoins           flexInt64 `json:"burned_coins"`
			RewardedCoins         flexInt64 `json:"rewarded_coins"`
		} `json:"current_epoch_stats"`
	} `json:"participant"`
}

func FetchParticipantStats(restURL, address string) (*ParticipantStats, error) {
	var r participantStatsResp
	if err := get(restURL+"/productscience/inference/inference/participant/"+address, &r); err != nil {
		return nil, err
	}
	p := r.Participant
	return &ParticipantStats{
		EpochsCompleted:              int64(p.EpochsCompleted),
		CoinBalance:                  int64(p.CoinBalance),
		InferenceCount:               int64(p.CurrentEpochStats.InferenceCount),
		MissedRequests:               int64(p.CurrentEpochStats.MissedRequests),
		EarnedCoins:                  int64(p.CurrentEpochStats.EarnedCoins),
		ValidatedInferences:          int64(p.CurrentEpochStats.ValidatedInferences),
		InvalidatedInferences:        int64(p.CurrentEpochStats.InvalidatedInferences),
		Status:                       p.Status,
		ConsecutiveInvalidInferences: int64(p.ConsecutiveInvalidInferences),
		BurnedCoins:                  int64(p.CurrentEpochStats.BurnedCoins),
		RewardedCoins:                int64(p.CurrentEpochStats.RewardedCoins),
	}, nil
}

// --- Bank balance ---

func FetchWalletBalance(restURL, address string) (float64, error) {
	var r struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := get(restURL+"/cosmos/bank/v1beta1/balances/"+address, &r); err != nil {
		return 0, err
	}
	for _, b := range r.Balances {
		if b.Denom == "ngonka" {
			n, err := strconv.ParseInt(b.Amount, 10, 64)
			if err != nil {
				return 0, err
			}
			return float64(n) / 1e9, nil
		}
	}
	return 0, fmt.Errorf("ngonka balance not found")
}

// --- Node list (admin API) ---

// NodeEntry represents a single node returned by the admin API.
type NodeEntry struct {
	Node struct {
		ID       string `json:"id"`
		Host     string `json:"host"`
		PocPort  int    `json:"poc_port"`
		Hardware []struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		} `json:"hardware"`
	} `json:"node"`
	State struct {
		CurrentStatus     string `json:"current_status"`
		IntendedStatus    string `json:"intended_status"`
		PocCurrentStatus  string `json:"poc_current_status"`
		PocIntendedStatus string `json:"poc_intended_status"`
		EpochMLNodes      map[string]struct {
			PocWeight          *int64 `json:"poc_weight"`
			TimeslotAllocation []bool `json:"timeslot_allocation"`
		} `json:"epoch_ml_nodes"`
	} `json:"state"`
}

func FetchNodes(adminURL string) ([]NodeEntry, error) {
	var nodes []NodeEntry
	err := get(adminURL+"/admin/v1/nodes", &nodes)
	return nodes, err
}

// --- GPU stats ---

// GPUDevice holds per-device GPU metrics.
type GPUDevice struct {
	Index              int
	UtilizationPercent float64
	TemperatureC       *float64
	TotalMemoryMB      *int64
	FreeMemoryMB       *int64
	UsedMemoryMB       *int64
	IsAvailable        bool
}

// GPUStats contains aggregated GPU info for a node.
type GPUStats struct {
	Count   int
	AvgUtil float64
	Devices []GPUDevice
}

func FetchGPUStats(host string, port int) GPUStats {
	var r struct {
		Devices []struct {
			Index              int      `json:"index"`
			UtilizationPercent float64  `json:"utilization_percent"`
			TemperatureC       *float64 `json:"temperature_c"`
			TotalMemoryMB      *int64   `json:"total_memory_mb"`
			FreeMemoryMB       *int64   `json:"free_memory_mb"`
			UsedMemoryMB       *int64   `json:"used_memory_mb"`
			IsAvailable        bool     `json:"is_available"`
		} `json:"devices"`
	}
	url := fmt.Sprintf("http://%s:%d/v3.0.8/api/v1/gpu/devices", host, port)
	if err := get(url, &r); err != nil || len(r.Devices) == 0 {
		return GPUStats{}
	}
	var total float64
	devices := make([]GPUDevice, len(r.Devices))
	for i, d := range r.Devices {
		total += d.UtilizationPercent
		devices[i] = GPUDevice{
			Index:              d.Index,
			UtilizationPercent: d.UtilizationPercent,
			TemperatureC:       d.TemperatureC,
			TotalMemoryMB:      d.TotalMemoryMB,
			FreeMemoryMB:       d.FreeMemoryMB,
			UsedMemoryMB:       d.UsedMemoryMB,
			IsAvailable:        d.IsAvailable,
		}
	}
	return GPUStats{Count: len(r.Devices), AvgUtil: total / float64(len(r.Devices)), Devices: devices}
}

// FetchMLNodeState returns the current service state string ("POW", "INFERENCE", "TRAIN", "STOPPED").
func FetchMLNodeState(host string, port int) (string, error) {
	var r struct {
		State string `json:"state"`
	}
	url := fmt.Sprintf("http://%s:%d/v3.0.8/api/v1/state", host, port)
	if err := get(url, &r); err != nil {
		return "", err
	}
	return r.State, nil
}

// FetchMLNodeDiskSpaceGB returns available disk space in GB for the model cache.
func FetchMLNodeDiskSpaceGB(host string, port int) (float64, error) {
	var r struct {
		AvailableGB float64 `json:"available_gb"`
	}
	url := fmt.Sprintf("http://%s:%d/v3.0.8/api/v1/models/space", host, port)
	if err := get(url, &r); err != nil {
		return 0, err
	}
	return r.AvailableGB, nil
}

// --- Participants (network-wide weights) ---

// ParticipantEntry is one entry in the active participants list.
type ParticipantEntry struct {
	Seed struct {
		Participant string `json:"participant"`
	} `json:"seed"`
	Weight  *float64 `json:"weight"`
	MLNodes []struct {
		MLNodes []struct {
			NodeID    string   `json:"node_id"`
			PocWeight *float64 `json:"poc_weight"`
		} `json:"ml_nodes"`
	} `json:"ml_nodes"`
}

func FetchNetworkParticipants(apiURL string) ([]ParticipantEntry, error) {
	var r struct {
		ActiveParticipants struct {
			Participants []ParticipantEntry `json:"participants"`
		} `json:"active_participants"`
	}
	if err := get(apiURL+"/v1/epochs/current/participants", &r); err != nil {
		return nil, err
	}
	return r.ActiveParticipants.Participants, nil
}

// --- Pricing ---

// PricingData holds current pricing configuration.
type PricingData struct {
	UnitOfComputePrice    *float64 `json:"unit_of_compute_price"`
	DynamicPricingEnabled *bool    `json:"dynamic_pricing_enabled"`
	Models []struct {
		ID                     string   `json:"id"`
		PricePerToken          *float64 `json:"price_per_token"`
		UnitsOfComputePerToken *float64 `json:"units_of_compute_per_token"`
		Utilization            *float64 `json:"utilization"`
		Capacity               *int64   `json:"capacity"`
	} `json:"models"`
}

func FetchPricing(apiURL string) (*PricingData, error) {
	var r PricingData
	err := get(apiURL+"/v1/pricing", &r)
	return &r, err
}

// --- Models ---

// ModelData holds model definitions from the API.
type ModelData struct {
	Models []struct {
		ID                  string   `json:"id"`
		VRAM                *float64 `json:"v_ram"`
		ThroughputPerNonce  *float64 `json:"throughput_per_nonce"`
		ValidationThreshold *struct {
			Value    float64 `json:"value"`
			Exponent int     `json:"exponent"`
		} `json:"validation_threshold"`
	} `json:"models"`
}

func FetchModels(apiURL string) (*ModelData, error) {
	var r ModelData
	err := get(apiURL+"/v1/models", &r)
	return &r, err
}

// BLSEpochData holds BLS DKG phase information for an epoch.
type BLSEpochData struct {
	DKGPhase                    int32
	DealingPhaseDeadlineBlock   int64
	VerifyingPhaseDeadlineBlock int64
}

// dkgPhaseNames maps DKG phase string names to numeric values.
var dkgPhaseNames = map[string]int32{
	"DKG_PHASE_UNDEFINED":  0,
	"DKG_PHASE_DEALING":    1,
	"DKG_PHASE_VERIFYING":  2,
	"DKG_PHASE_COMPLETED":  3,
	"DKG_PHASE_FAILED":     4,
	"DKG_PHASE_SIGNED":     5,
}

// --- Stats API ---

// StatsSummaryData holds network-wide inference statistics.
type StatsSummaryData struct {
	AiTokens   int64
	Inferences int32
	ActualCost int64
}

// FetchStatsSummary returns aggregated network stats from /v1/stats/summary/time.
func FetchStatsSummary(apiURL string) (*StatsSummaryData, error) {
	var r struct {
		AiTokens             int64 `json:"ai_tokens"`
		Inferences           int32 `json:"inferences"`
		ActualInferencesCost int64 `json:"actual_inferences_cost"`
	}
	if err := get(apiURL+"/v1/stats/summary/time", &r); err != nil {
		return nil, err
	}
	return &StatsSummaryData{AiTokens: r.AiTokens, Inferences: r.Inferences, ActualCost: r.ActualInferencesCost}, nil
}

// StatsModelEntry holds per-model inference statistics.
type StatsModelEntry struct {
	Model      string
	AiTokens   int64
	Inferences int32
}

// FetchStatsModels returns per-model stats from /v1/stats/models.
func FetchStatsModels(apiURL string) ([]StatsModelEntry, error) {
	var r struct {
		StatsModels []struct {
			Model      string `json:"model"`
			AiTokens   int64  `json:"ai_tokens"`
			Inferences int32  `json:"inferences"`
		} `json:"stats_models"`
	}
	if err := get(apiURL+"/v1/stats/models", &r); err != nil {
		return nil, err
	}
	out := make([]StatsModelEntry, len(r.StatsModels))
	for i, m := range r.StatsModels {
		out[i] = StatsModelEntry{Model: m.Model, AiTokens: m.AiTokens, Inferences: m.Inferences}
	}
	return out, nil
}

// --- Bridge ---

// BridgeStatusData holds bridge queue status.
type BridgeStatusData struct {
	PendingBlocks   int
	PendingReceipts int
	ReadyToProcess  bool
	EarliestBlock   uint64
	LatestBlock     uint64
}

// FetchBridgeStatus returns bridge queue status from /v1/bridge/status.
func FetchBridgeStatus(apiURL string) (*BridgeStatusData, error) {
	var r struct {
		PendingBlocksCount   int    `json:"pendingBlocksCount"`
		PendingReceiptsCount int    `json:"pendingReceiptsCount"`
		EarliestBlockNumber  uint64 `json:"earliestBlockNumber"`
		LatestBlockNumber    uint64 `json:"latestBlockNumber"`
		ReadyToProcess       bool   `json:"readyToProcess"`
	}
	if err := get(apiURL+"/v1/bridge/status", &r); err != nil {
		return nil, err
	}
	return &BridgeStatusData{
		PendingBlocks:   r.PendingBlocksCount,
		PendingReceipts: r.PendingReceiptsCount,
		ReadyToProcess:  r.ReadyToProcess,
		EarliestBlock:   r.EarliestBlockNumber,
		LatestBlock:     r.LatestBlockNumber,
	}, nil
}

// --- ML node health ---

// MLNodeManagerStatus holds running/healthy status for one manager.
type MLNodeManagerStatus struct {
	Running bool
	Healthy bool
}

// MLNodeHealthData holds manager health for all ML node managers.
type MLNodeHealthData struct {
	ManagerPow       MLNodeManagerStatus
	ManagerInference MLNodeManagerStatus
	ManagerTrain     MLNodeManagerStatus
}

// FetchMLNodeHealth returns manager health. Tries /health first (root endpoint),
// then /v3.0.8/api/v1/health as fallback.
func FetchMLNodeHealth(host string, port int) (*MLNodeHealthData, error) {
	var r struct {
		Managers struct {
			Pow struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"pow"`
			Inference struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"inference"`
			Train struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"train"`
		} `json:"managers"`
	}
	url := fmt.Sprintf("http://%s:%d/health", host, port)
	if err := get(url, &r); err != nil {
		// Fallback: versioned path
		url = fmt.Sprintf("http://%s:%d/v3.0.8/api/v1/health", host, port)
		if err2 := get(url, &r); err2 != nil {
			return nil, err2
		}
	}
	return &MLNodeHealthData{
		ManagerPow:       MLNodeManagerStatus{Running: r.Managers.Pow.Running, Healthy: r.Managers.Pow.Healthy},
		ManagerInference: MLNodeManagerStatus{Running: r.Managers.Inference.Running, Healthy: r.Managers.Inference.Healthy},
		ManagerTrain:     MLNodeManagerStatus{Running: r.Managers.Train.Running, Healthy: r.Managers.Train.Healthy},
	}, nil
}

// --- GPU driver info ---

// GPUDriverData holds GPU driver and CUDA version strings.
type GPUDriverData struct {
	DriverVersion     string
	CudaDriverVersion string
}

// FetchGPUDriverInfo returns GPU driver info from /v3.0.8/api/v1/gpu/driver.
func FetchGPUDriverInfo(host string, port int) (*GPUDriverData, error) {
	var r struct {
		DriverVersion     string `json:"driver_version"`
		CudaDriverVersion string `json:"cuda_driver_version"`
	}
	url := fmt.Sprintf("http://%s:%d/v3.0.8/api/v1/gpu/driver", host, port)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	if r.DriverVersion == "" {
		return nil, fmt.Errorf("empty driver version")
	}
	return &GPUDriverData{DriverVersion: r.DriverVersion, CudaDriverVersion: r.CudaDriverVersion}, nil
}

// --- Tokenomics ---

// TokenomicsData holds chain-wide tokenomics counters.
type TokenomicsData struct {
	TotalFees      uint64
	TotalSubsidies uint64
	TotalRefunded  uint64
	TotalBurned    uint64
}

// FetchTokenomics returns tokenomics data from the chain REST endpoint.
func FetchTokenomics(restURL string) (*TokenomicsData, error) {
	var r struct {
		TokenomicsData struct {
			TotalFees      flexInt64 `json:"total_fees"`
			TotalSubsidies flexInt64 `json:"total_subsidies"`
			TotalRefunded  flexInt64 `json:"total_refunded"`
			TotalBurned    flexInt64 `json:"total_burned"`
		} `json:"tokenomics_data"`
	}
	if err := get(restURL+"/productscience/inference/inference/tokenomics_data", &r); err != nil {
		return nil, err
	}
	td := r.TokenomicsData
	return &TokenomicsData{
		TotalFees:      uint64(td.TotalFees),
		TotalSubsidies: uint64(td.TotalSubsidies),
		TotalRefunded:  uint64(td.TotalRefunded),
		TotalBurned:    uint64(td.TotalBurned),
	}, nil
}

// --- PoC v2 ---

// PoCv2CommitData holds the artifact commit count for a participant.
type PoCv2CommitData struct {
	Count uint32
}

// FetchPoCv2Commit returns the PoC v2 store commit for a participant at a given PoC stage start block.
func FetchPoCv2Commit(restURL string, pocStartBlock int64, address string) (*PoCv2CommitData, error) {
	var r struct {
		PoCV2StoreCommit struct {
			Count flexInt64 `json:"count"`
		} `json:"poc_v2_store_commit"`
	}
	url := fmt.Sprintf("%s/productscience/inference/inference/poc_v2_store_commit/%d/%s", restURL, pocStartBlock, address)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	return &PoCv2CommitData{Count: uint32(r.PoCV2StoreCommit.Count)}, nil
}

// MLNodeWeightEntry holds per-node weight from MLNode weight distribution.
type MLNodeWeightEntry struct {
	NodeID string
	Weight uint32
}

// FetchMLNodeWeightDist returns the per-node weight distribution for a participant at a given PoC stage start block.
func FetchMLNodeWeightDist(restURL string, pocStartBlock int64, address string) ([]MLNodeWeightEntry, error) {
	var r struct {
		MLNodeWeightDistribution struct {
			Weights []struct {
				NodeID string    `json:"node_id"`
				Weight flexInt64 `json:"weight"`
			} `json:"weights"`
		} `json:"mlnode_weight_distribution"`
	}
	url := fmt.Sprintf("%s/productscience/inference/inference/mlnode_weight_distribution/%d/%s", restURL, pocStartBlock, address)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	out := make([]MLNodeWeightEntry, len(r.MLNodeWeightDistribution.Weights))
	for i, w := range r.MLNodeWeightDistribution.Weights {
		out[i] = MLNodeWeightEntry{NodeID: w.NodeID, Weight: uint32(w.Weight)}
	}
	return out, nil
}

// FetchBLSEpoch fetches BLS DKG phase data for the given epoch ID from the public API.
func FetchBLSEpoch(apiURL string, epochID int64) (*BLSEpochData, error) {
	var r struct {
		EpochData struct {
			DKGPhase                    interface{} `json:"dkg_phase"`
			DealingPhaseDeadlineBlock   flexInt64   `json:"dealing_phase_deadline_block"`
			VerifyingPhaseDeadlineBlock flexInt64   `json:"verifying_phase_deadline_block"`
		} `json:"epoch_data"`
	}
	url := fmt.Sprintf("%s/v1/bls/epoch/%d", apiURL, epochID)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	var phase int32
	switch v := r.EpochData.DKGPhase.(type) {
	case float64:
		phase = int32(v)
	case string:
		if n, ok := dkgPhaseNames[v]; ok {
			phase = n
		}
	}
	return &BLSEpochData{
		DKGPhase:                    phase,
		DealingPhaseDeadlineBlock:   int64(r.EpochData.DealingPhaseDeadlineBlock),
		VerifyingPhaseDeadlineBlock: int64(r.EpochData.VerifyingPhaseDeadlineBlock),
	}, nil
}
