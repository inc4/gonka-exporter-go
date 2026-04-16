package fetcher

import (
	"fmt"
	"strconv"
)

func (h *HTTPFetcher) FetchCurrentEpoch(restURL string) (int64, error) {
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

func (h *HTTPFetcher) FetchEpochInfo(restURL string) (*EpochInfo, error) {
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
	el, err2 := strconv.ParseInt(r.Params.EpochParams.EpochLength, 10, 64)
	bh, err3 := strconv.ParseInt(r.BlockHeight, 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, fmt.Errorf("parse epoch_info: %v %v %v", err1, err2, err3)
	}
	return &EpochInfo{
		PocStartBlockHeight: poc,
		EpochLength:         el,
		BlockHeight:         bh,
	}, nil
}

// ValidationWeight is a per-participant weight entry.
type ValidationWeight struct {
	MemberAddress string    `json:"member_address"`
	Weight        string    `json:"weight"`
	Reputation    flexInt64 `json:"reputation"`
}

// EpochGroupData contains network-wide epoch weight data.
type EpochGroupData struct {
	TotalWeight      int64
	EpochIndex       int64
	NumberOfRequests int64
	ValidationWeights []ValidationWeight
}

func (h *HTTPFetcher) FetchEpochGroupData(restURL string) (*EpochGroupData, error) {
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

// EpochPerfSummary holds on-chain reward data for a completed epoch.
type EpochPerfSummary struct {
	RewardedGNK float64
	Claimed     int
}

func (h *HTTPFetcher) FetchEpochPerfSummary(restURL, address string, epochNum int64) (*EpochPerfSummary, error) {
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

func (h *HTTPFetcher) FetchParticipantStats(restURL, address string) (*ParticipantStats, error) {
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

func (h *HTTPFetcher) FetchWalletBalance(restURL, address string) (float64, error) {
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
	return 0, nil // new wallet with zero balance — valid state, not an error
}

// TokenomicsData holds chain-wide tokenomics counters.
type TokenomicsData struct {
	TotalFees      uint64
	TotalSubsidies uint64
	TotalRefunded  uint64
	TotalBurned    uint64
}

// FetchTokenomics returns tokenomics data from the chain REST endpoint.
func (h *HTTPFetcher) FetchTokenomics(restURL string) (*TokenomicsData, error) {
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

// PoCv2CommitData holds the artifact commit count for a participant.
type PoCv2CommitData struct {
	Count uint32
}

// FetchPoCv2Commit returns the PoC v2 store commit for a participant at a given PoC stage start block.
func (h *HTTPFetcher) FetchPoCv2Commit(restURL string, pocStartBlock int64, address string) (*PoCv2CommitData, error) {
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
func (h *HTTPFetcher) FetchMLNodeWeightDist(restURL string, pocStartBlock int64, address string) ([]MLNodeWeightEntry, error) {
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
