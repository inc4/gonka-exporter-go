package fetcher

import "fmt"

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

func (h *HTTPFetcher) FetchNetworkParticipants(apiURL string) ([]ParticipantEntry, error) {
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

// PricingData holds current pricing configuration.
type PricingData struct {
	UnitOfComputePrice    *float64 `json:"unit_of_compute_price"`
	DynamicPricingEnabled *bool    `json:"dynamic_pricing_enabled"`
	Models                []struct {
		ID                     string   `json:"id"`
		PricePerToken          *float64 `json:"price_per_token"`
		UnitsOfComputePerToken *float64 `json:"units_of_compute_per_token"`
		Utilization            *float64 `json:"utilization"`
		Capacity               *int64   `json:"capacity"`
	} `json:"models"`
}

func (h *HTTPFetcher) FetchPricing(apiURL string) (*PricingData, error) {
	var r PricingData
	err := get(apiURL+"/v1/pricing", &r)
	return &r, err
}

// ModelData holds model definitions from the API.
type ModelData struct {
	Models []struct {
		ID                 string   `json:"id"`
		VRAM               *float64 `json:"v_ram"`
		ThroughputPerNonce *float64 `json:"throughput_per_nonce"`
		ValidationThreshold *struct {
			Value    float64 `json:"value"`
			Exponent int     `json:"exponent"`
		} `json:"validation_threshold"`
	} `json:"models"`
}

func (h *HTTPFetcher) FetchModels(apiURL string) (*ModelData, error) {
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
	"DKG_PHASE_UNDEFINED": 0,
	"DKG_PHASE_DEALING":   1,
	"DKG_PHASE_VERIFYING": 2,
	"DKG_PHASE_COMPLETED": 3,
	"DKG_PHASE_FAILED":    4,
	"DKG_PHASE_SIGNED":    5,
}

// FetchBLSEpoch fetches BLS DKG phase data for the given epoch ID from the public API.
func (h *HTTPFetcher) FetchBLSEpoch(apiURL string, epochID int64) (*BLSEpochData, error) {
	var r struct {
		EpochData struct {
			DKGPhase                    any       `json:"dkg_phase"`
			DealingPhaseDeadlineBlock   flexInt64 `json:"dealing_phase_deadline_block"`
			VerifyingPhaseDeadlineBlock flexInt64 `json:"verifying_phase_deadline_block"`
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

// StatsSummaryData holds network-wide inference statistics.
type StatsSummaryData struct {
	AiTokens   int64
	Inferences int32
	ActualCost int64
}

// FetchStatsSummary returns aggregated network stats from /v1/stats/summary/time.
func (h *HTTPFetcher) FetchStatsSummary(apiURL string) (*StatsSummaryData, error) {
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
func (h *HTTPFetcher) FetchStatsModels(apiURL string) ([]StatsModelEntry, error) {
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

// BridgeStatusData holds bridge queue status.
type BridgeStatusData struct {
	PendingBlocks   int
	PendingReceipts int
	ReadyToProcess  bool
	EarliestBlock   uint64
	LatestBlock     uint64
}

// FetchBridgeStatus returns bridge queue status from /v1/bridge/status.
func (h *HTTPFetcher) FetchBridgeStatus(apiURL string) (*BridgeStatusData, error) {
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
