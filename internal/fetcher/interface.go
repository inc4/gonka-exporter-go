package fetcher

// Fetcher defines all upstream API calls used by the collector.
// HTTPFetcher is the production implementation; tests can inject a mock.
type Fetcher interface {
	FetchTendermintStatus(rpcURL string) (*TendermintStatus, error)
	FetchBlockTimeAtHeight(rpcURL string, height int64) (float64, error)
	FetchMaxBlockHeightFromNodes(nodes []string) (int64, string)

	FetchCurrentEpoch(restURL string) (int64, error)
	FetchEpochInfo(restURL string) (*EpochInfo, error)
	FetchEpochGroupData(restURL string) (*EpochGroupData, error)
	FetchEpochPerfSummary(restURL, address string, epochNum int64) (*EpochPerfSummary, error)
	FetchBLSEpoch(apiURL string, epochID int64) (*BLSEpochData, error)

	FetchParticipantStats(restURL, address string) (*ParticipantStats, error)
	FetchWalletBalance(restURL, address string) (float64, error)

	FetchNetworkParticipants(apiURL string) ([]ParticipantEntry, error)
	FetchPricing(apiURL string) (*PricingData, error)
	FetchModels(apiURL string) (*ModelData, error)
	FetchStatsSummary(apiURL string) (*StatsSummaryData, error)
	FetchStatsModels(apiURL string) ([]StatsModelEntry, error)
	FetchBridgeStatus(apiURL string) (*BridgeStatusData, error)

	FetchNodes(adminURL string) ([]NodeEntry, error)
	FetchGPUStats(host string, port int) GPUStats
	FetchMLNodeState(host string, port int) (string, error)
	FetchMLNodeDiskSpaceGB(host string, port int) (float64, error)
	FetchGPUDriverInfo(host string, port int) (*GPUDriverData, error)
	FetchMLNodeHealth(host string, port int) (*MLNodeHealthData, error)

	FetchTokenomics(restURL string) (*TokenomicsData, error)
	FetchPoCv2Commit(restURL string, pocStartBlock int64, address string) (*PoCv2CommitData, error)
	FetchMLNodeWeightDist(restURL string, pocStartBlock int64, address string) ([]MLNodeWeightEntry, error)
}

// HTTPFetcher is the production implementation of Fetcher using real HTTP calls.
type HTTPFetcher struct{}

// NewHTTPFetcher returns a Fetcher backed by real HTTP calls.
func NewHTTPFetcher() *HTTPFetcher { return &HTTPFetcher{} }
