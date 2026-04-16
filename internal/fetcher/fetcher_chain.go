package fetcher

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TendermintStatus is the response from the Tendermint /status RPC endpoint.
type TendermintStatus struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			LatestBlockTime   string `json:"latest_block_time"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

func (h *HTTPFetcher) FetchTendermintStatus(rpcURL string) (*TendermintStatus, error) {
	var s TendermintStatus
	err := get(rpcURL+"/status", &s)
	return &s, err
}

func (h *HTTPFetcher) FetchBlockTimeAtHeight(rpcURL string, height int64) (float64, error) {
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

func (h *HTTPFetcher) FetchMaxBlockHeightFromNodes(nodes []string) (int64, string) {
	if len(nodes) == 0 {
		return 0, ""
	}
	sample := rand.Perm(len(nodes))
	if len(sample) > 5 {
		sample = sample[:5]
	}

	var mu sync.Mutex
	var maxHeight int64
	var latestTime string

	var wg sync.WaitGroup
	for _, idx := range sample {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()
			var resp struct {
				Result struct {
					SyncInfo struct {
						LatestBlockHeight string `json:"latest_block_height"`
						LatestBlockTime   string `json:"latest_block_time"`
					} `json:"sync_info"`
				} `json:"result"`
			}
			if err := get(nodeURL+"/chain-rpc/status", &resp); err != nil {
				return
			}
			height, err := strconv.ParseInt(resp.Result.SyncInfo.LatestBlockHeight, 10, 64)
			if err != nil {
				return
			}
			mu.Lock()
			if height > maxHeight {
				maxHeight = height
				latestTime = resp.Result.SyncInfo.LatestBlockTime
			}
			mu.Unlock()
		}(nodes[idx])
	}
	wg.Wait()
	return maxHeight, latestTime
}
