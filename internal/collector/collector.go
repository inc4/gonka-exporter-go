package collector

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gonka/exporter/internal/config"
	"github.com/gonka/exporter/internal/fetcher"
	"github.com/gonka/exporter/internal/metrics"
	"github.com/gonka/exporter/internal/state"
)

var (
	nodeStatusMap = map[string]float64{
		"UNKNOWN": 0, "INFERENCE": 1, "POC": 2, "TRAINING": 3, "STOPPED": 4, "FAILED": 5,
	}
	pocStatusMap = map[string]float64{
		"IDLE": 0, "GENERATING": 1, "VALIDATING": 2,
	}
	participantStatusMap = map[string]float64{
		"UNSPECIFIED": 0, "ACTIVE": 1, "INACTIVE": 2, "INVALID": 3, "UNCONFIRMED": 5,
	}
	mlNodeServiceStateMap = map[string]float64{
		"STOPPED": 0, "INFERENCE": 1, "POW": 2, "TRAIN": 3,
	}
)

// Collector holds all mutable state for one collection cycle.
type Collector struct {
	cfg             config.Config
	f               fetcher.Fetcher
	m               *metrics.Metrics
	st              state.EpochState
	history         state.History
	epochMax        map[int64]*state.EpochMaxValues
	epochNode       state.EpochNodeState
	estimatedReward float64 // last calculated estimated reward for current epoch
}

// New creates a Collector, restoring persisted state and history.
// reg is the Prometheus registerer to use; pass prometheus.DefaultRegisterer in production,
// prometheus.NewRegistry() in tests.
func New(cfg config.Config, f fetcher.Fetcher, reg prometheus.Registerer) *Collector {
	st := state.LoadState(cfg.StateFile)
	history := state.LoadHistory(cfg.HistoryFile)
	c := &Collector{
		cfg:       cfg,
		f:         f,
		m:         metrics.NewMetrics(reg),
		st:        st,
		history:   history,
		epochMax:  make(map[int64]*state.EpochMaxValues),
		epochNode: state.NewEpochNodeState(),
	}
	c.restoreMetrics()
	return c
}

// Collect runs one full collection cycle.
func (c *Collector) Collect() {
	slog.Info("collecting metrics", "participant", c.cfg.Participant)
	c.collectChain()
	c.collectNetworkParticipants()
	c.collectPricingAndModels()
	c.collectParticipant()
	c.collectNodes()
	c.collectStats()
	c.collectBridge()
	c.collectTokenomics()
	c.collectPoCv2()
}

// --- Chain / block ---

func (c *Collector) collectChain() {
	p := c.cfg.Participant
	if p == "" {
		p = "unknown"
	}

	go func() {
		h, t := c.f.FetchMaxBlockHeightFromNodes(c.cfg.BlockHeightNodes)
		if h > 0 {
			c.m.BlockHeightMax.WithLabelValues(p).Set(float64(h))
		}
		if t != "" {
			ts := strings.TrimSuffix(t, "Z")
			if parsed, err := time.Parse("2006-01-02T15:04:05.999999999", ts); err == nil {
				c.m.BlockTimeNetwork.WithLabelValues(p).Set(float64(parsed.Unix()))
			}
		}
	}()

	status, err := c.f.FetchTendermintStatus(c.cfg.NodeRPCURL)
	if err != nil {
		slog.Warn("tendermint status", "err", err)
		return
	}
	si := status.Result.SyncInfo
	if h, err := strconv.ParseInt(si.LatestBlockHeight, 10, 64); err == nil {
		c.m.BlockHeight.WithLabelValues(p).Set(float64(h))
	}
	if si.LatestBlockTime != "" {
		ts := strings.TrimSuffix(si.LatestBlockTime, "Z")
		if parsed, err := time.Parse("2006-01-02T15:04:05.999999999", ts); err == nil {
			c.m.BlockTimeLocal.WithLabelValues(p).Set(float64(parsed.Unix()))
		}
	}
	catching := 0.0
	if si.CatchingUp {
		catching = 1.0
	}
	c.m.CatchingUp.WithLabelValues(p).Set(catching)
}

// --- Network-wide participants ---

func (c *Collector) collectNetworkParticipants() {
	participants, err := c.f.FetchNetworkParticipants(c.cfg.APIURL)
	if err != nil {
		slog.Warn("network participants", "err", err)
		return
	}
	c.m.NetTotalParticipantCount.Set(float64(len(participants)))
	c.m.NetParticipantWeight.Reset()
	c.m.NetNodePocWeight.Reset()
	active := 0
	for _, p := range participants {
		addr := p.Seed.Participant
		if addr == "" {
			continue
		}
		active++
		if p.Weight != nil {
			c.m.NetParticipantWeight.WithLabelValues(addr).Set(*p.Weight)
		}
		for _, group := range p.MLNodes {
			for _, node := range group.MLNodes {
				if node.NodeID != "" && node.PocWeight != nil {
					c.m.NetNodePocWeight.WithLabelValues(addr, node.NodeID).Set(*node.PocWeight)
				}
			}
		}
	}
	c.m.NetActiveParticipantCount.Set(float64(active))
}

// --- Pricing and models ---

func (c *Collector) collectPricingAndModels() {
	pricing, err := c.f.FetchPricing(c.cfg.APIURL)
	if err != nil {
		slog.Warn("pricing", "err", err)
	} else {
		if pricing.UnitOfComputePrice != nil {
			c.m.PricingUoC.Set(*pricing.UnitOfComputePrice)
		}
		if pricing.DynamicPricingEnabled != nil {
			v := 0.0
			if *pricing.DynamicPricingEnabled {
				v = 1.0
			}
			c.m.PricingDynamic.Set(v)
		}
		for _, m := range pricing.Models {
			if m.ID == "" {
				continue
			}
			if m.PricePerToken != nil {
				c.m.ModelPrice.WithLabelValues(m.ID).Set(*m.PricePerToken)
			}
			if m.UnitsOfComputePerToken != nil {
				c.m.ModelUnits.WithLabelValues(m.ID).Set(*m.UnitsOfComputePerToken)
			}
			if m.Utilization != nil {
				c.m.ModelUtilization.WithLabelValues(m.ID).Set(*m.Utilization * 100)
			}
			if m.Capacity != nil {
				c.m.ModelCapacity.WithLabelValues(m.ID).Set(float64(*m.Capacity))
			}
		}
	}

	models, err := c.f.FetchModels(c.cfg.APIURL)
	if err != nil {
		slog.Warn("models", "err", err)
		return
	}
	for _, m := range models.Models {
		if m.ID == "" {
			continue
		}
		if m.VRAM != nil {
			c.m.ModelVRAM.WithLabelValues(m.ID).Set(*m.VRAM)
		}
		if m.ThroughputPerNonce != nil {
			c.m.ModelThroughput.WithLabelValues(m.ID).Set(*m.ThroughputPerNonce)
		}
		if m.ValidationThreshold != nil {
			c.m.ModelValThresh.WithLabelValues(m.ID).Set(
				m.ValidationThreshold.Value * math.Pow10(m.ValidationThreshold.Exponent),
			)
		}
	}
}

// --- Participant (main logic with epoch boundary detection) ---

func (c *Collector) collectParticipant() {
	addr := c.cfg.Participant
	if addr == "" {
		return
	}

	now := float64(time.Now().UnixNano()) / 1e9
	prevEpochStartTime := c.st.EpochStartTime
	chainEpoch := c.fetchAndSetChainEpoch(addr)

	epochInfo := c.fetchAndSetEpochInfo(chainEpoch)

	c.fetchAndSetGroupData(addr, chainEpoch)

	c.fetchAndSetBLS(addr, chainEpoch)

	stats, ok := c.fetchAndSetParticipantStats(addr)
	if !ok {
		return
	}

	wallet, walletOK := c.fetchAndSetWallet(addr)

	c.updateEpochMaxAndGauges(addr, chainEpoch, stats, wallet, walletOK, epochInfo, now)

	c.detectAndRecordEpochBoundary(chainEpoch, prevEpochStartTime, wallet, walletOK)
}

// --- collectParticipant helpers ---

func (c *Collector) detectAndRecordEpochBoundary(chainEpoch int64, prevEpochStartTime float64, wallet float64, walletOK bool) {
	prevEpoch := c.st.ChainEpoch
	if c.st.Valid && prevEpoch > 0 && chainEpoch > prevEpoch {
		if chainEpoch > prevEpoch+1 {
			slog.Warn("skipped epochs", "from", prevEpoch, "to", chainEpoch)
		}
		c.recordSnapshot(prevEpoch, prevEpochStartTime, c.st.EpochStartTime, wallet, walletOK)
		if walletOK {
			c.st.WalletBalanceAtEpochStart = wallet
		}
		c.pruneEpochMax()
	} else if !c.st.Valid {
		if walletOK {
			c.st.WalletBalanceAtEpochStart = wallet
		}
	}
	changed := c.st.ChainEpoch != chainEpoch
	c.st.ChainEpoch = chainEpoch
	c.st.Valid = true
	if changed {
		state.SaveState(c.cfg.StateFile, c.st)
	}
}

func (c *Collector) updateEpochMaxAndGauges(addr string, chainEpoch int64, stats *fetcher.ParticipantStats, wallet float64, walletOK bool, epochInfo *fetcher.EpochInfo, now float64) {
	if chainEpoch == 0 {
		return
	}
	if _, ok := c.epochMax[chainEpoch]; !ok {
		c.epochMax[chainEpoch] = &state.EpochMaxValues{}
	}
	em := c.epochMax[chainEpoch]
	em.InferenceCount        = max(em.InferenceCount,        stats.InferenceCount)
	em.MissedRequests        = max(em.MissedRequests,        stats.MissedRequests)
	em.EarnedCoins           = max(em.EarnedCoins,           stats.EarnedCoins)
	em.ValidatedInferences   = max(em.ValidatedInferences,   stats.ValidatedInferences)
	em.InvalidatedInferences = max(em.InvalidatedInferences, stats.InvalidatedInferences)
	em.CoinBalance           = max(em.CoinBalance,           stats.CoinBalance)
	em.EpochsCompleted       = max(em.EpochsCompleted,       stats.EpochsCompleted)

	ce := strconv.FormatInt(chainEpoch, 10)
	c.m.EpochInferences.WithLabelValues(addr, ce).Set(float64(em.InferenceCount))
	c.m.EpochMissed.WithLabelValues(addr, ce).Set(float64(em.MissedRequests))
	c.m.EpochEarnedCoins.WithLabelValues(addr, ce).Set(float64(em.EarnedCoins))
	c.m.EpochValidated.WithLabelValues(addr, ce).Set(float64(em.ValidatedInferences))
	c.m.EpochInvalidated.WithLabelValues(addr, ce).Set(float64(em.InvalidatedInferences))
	c.m.EpochCoinBalance.WithLabelValues(addr, ce).Set(float64(em.CoinBalance))
	c.m.EpochDone.WithLabelValues(addr, ce).Set(float64(em.EpochsCompleted))
	c.m.EpochMissRate.WithLabelValues(addr, ce).Set(state.MissRate(em.InferenceCount, em.MissedRequests))

	if walletOK {
		c.m.EpochEarnedGNK.WithLabelValues(addr, ce).Set(wallet - c.st.WalletBalanceAtEpochStart)
	}
	if c.st.EpochStartTime > 0 {
		c.m.EpochStartTime.WithLabelValues(addr, ce).Set(c.st.EpochStartTime)
	}
	if epochInfo != nil && c.st.EpochStartTime > 0 && c.st.PocStartBlockHeight > 0 && c.st.EpochLength > 0 {
		blocksElapsed := epochInfo.BlockHeight - c.st.PocStartBlockHeight
		timeElapsed := now - c.st.EpochStartTime
		if blocksElapsed > 0 && timeElapsed > 0 {
			avgBlockTime := timeElapsed / float64(blocksElapsed)
			blocksRemaining := c.st.EpochLength - blocksElapsed
			c.m.EpochEndTime.WithLabelValues(addr, ce).Set(now + float64(blocksRemaining)*avgBlockTime)
		}
	}
}

func (c *Collector) fetchAndSetWallet(addr string) (float64, bool) {
	wallet, err := c.f.FetchWalletBalance(c.cfg.NodeRESTURL, addr)
	if err != nil {
		slog.Warn("fetch wallet balance", "err", err)
		return 0, false
	}
	c.m.ParticipantWallet.WithLabelValues(addr).Set(wallet)
	if c.st.WalletBalanceAtEpochStart == 0 {
		c.st.WalletBalanceAtEpochStart = wallet
	}
	return wallet, true
}

func (c *Collector) fetchAndSetParticipantStats(addr string) (*fetcher.ParticipantStats, bool) {
	stats, err := c.f.FetchParticipantStats(c.cfg.NodeRESTURL, addr)
	if err != nil {
		slog.Warn("fetch participant stats", "err", err)
		return nil, false
	}
	c.m.ParticipantEpochsDone.WithLabelValues(addr).Set(float64(stats.EpochsCompleted))
	c.m.ParticipantCoinBalance.WithLabelValues(addr).Set(float64(stats.CoinBalance))
	c.m.ParticipantInferences.WithLabelValues(addr).Set(float64(stats.InferenceCount))
	c.m.ParticipantMissed.WithLabelValues(addr).Set(float64(stats.MissedRequests))
	c.m.ParticipantEarnedCoins.WithLabelValues(addr).Set(float64(stats.EarnedCoins))
	c.m.ParticipantValidated.WithLabelValues(addr).Set(float64(stats.ValidatedInferences))
	c.m.ParticipantInvalidated.WithLabelValues(addr).Set(float64(stats.InvalidatedInferences))
	if sv, ok := participantStatusMap[stats.Status]; ok {
		c.m.ParticipantStatus.WithLabelValues(addr).Set(sv)
	}
	c.m.ParticipantConsecutiveInv.WithLabelValues(addr).Set(float64(stats.ConsecutiveInvalidInferences))
	c.m.ParticipantBurnedCoins.WithLabelValues(addr).Set(float64(stats.BurnedCoins))
	c.m.ParticipantRewardedCoins.WithLabelValues(addr).Set(float64(stats.RewardedCoins))
	return stats, true
}

func (c *Collector) fetchAndSetBLS(addr string, chainEpoch int64) {
	if chainEpoch == 0 {
		return
	}
	bls, err := c.f.FetchBLSEpoch(c.cfg.APIURL, chainEpoch)
	if err != nil {
		slog.Warn("fetch bls epoch", "err", err)
		return
	}
	c.m.BLSDKGPhase.WithLabelValues(addr).Set(float64(bls.DKGPhase))
	if bls.DealingPhaseDeadlineBlock > 0 {
		c.m.BLSDealingDeadline.WithLabelValues(addr).Set(float64(bls.DealingPhaseDeadlineBlock))
	}
	if bls.VerifyingPhaseDeadlineBlock > 0 {
		c.m.BLSVerifyingDeadline.WithLabelValues(addr).Set(float64(bls.VerifyingPhaseDeadlineBlock))
	}
}

func (c *Collector) fetchAndSetGroupData(addr string, chainEpoch int64) {
	groupData, err := c.f.FetchEpochGroupData(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch epoch group data", "err", err)
		return
	}
	if groupData.NumberOfRequests > 0 {
		c.m.NetEpochInferenceCount.WithLabelValues(addr).Set(float64(groupData.NumberOfRequests))
	}
	if groupData.TotalWeight > 0 {
		// emission(N) = 323000 × exp(-0.000475 × (N-1))
		emission := 323000.0 * math.Exp(-0.000475*float64(groupData.EpochIndex-1))
		rewardPerWeight := emission / float64(groupData.TotalWeight)
		c.m.NetTotalWeight.WithLabelValues(addr).Set(float64(groupData.TotalWeight))
		c.m.NetRewardPerWeight.WithLabelValues(addr).Set(rewardPerWeight)
		for _, vw := range groupData.ValidationWeights {
			if vw.MemberAddress == addr {
				myWeight, _ := strconv.ParseInt(vw.Weight, 10, 64)
				c.estimatedReward = float64(myWeight) * rewardPerWeight
				c.m.ParticipantReputation.WithLabelValues(addr).Set(float64(vw.Reputation))
				if chainEpoch > 0 {
					ce := strconv.FormatInt(chainEpoch, 10)
					c.m.EpochEstimated.WithLabelValues(addr, ce).Set(c.estimatedReward)
				}
				break
			}
		}
	}
}

func (c *Collector) fetchAndSetEpochInfo(chainEpoch int64) *fetcher.EpochInfo {
	epochInfo, err := c.f.FetchEpochInfo(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch epoch info", "err", err)
		return nil
	}
	c.st.EpochLength = epochInfo.EpochLength
	if epochInfo.PocStartBlockHeight != c.st.PocStartBlockHeight {
		t, err := c.f.FetchBlockTimeAtHeight(c.cfg.NodeRPCURL, epochInfo.PocStartBlockHeight)
		if err != nil {
			slog.Warn("fetch block time", "height", epochInfo.PocStartBlockHeight, "err", err)
		} else {
			c.st.PocStartBlockHeight = epochInfo.PocStartBlockHeight
			c.st.EpochStartTime = t
			slog.Info("epoch start block time",
				"epoch", chainEpoch,
				"block", epochInfo.PocStartBlockHeight,
				"time", time.Unix(int64(t), 0).UTC().Format(time.RFC3339))
		}
	}
	return epochInfo
}

func (c *Collector) fetchAndSetChainEpoch(addr string) int64 {
	chainEpoch, err := c.f.FetchCurrentEpoch(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch current epoch", "err", err)
		return 0
	}
	c.m.ChainEpoch.WithLabelValues(addr).Set(float64(chainEpoch))
	return chainEpoch
}

func (c *Collector) recordSnapshot(epoch int64, startTime, endTime float64, wallet float64, walletOK bool) {
	addr := c.cfg.Participant
	em := c.epochMax[epoch]
	if em == nil {
		em = &state.EpochMaxValues{}
	}

	duration := endTime - startTime
	snap := &state.EpochSnapshot{
		Participant:           addr,
		InferenceCount:        em.InferenceCount,
		MissedRequests:        em.MissedRequests,
		EarnedCoins:           em.EarnedCoins,
		ValidatedInferences:   em.ValidatedInferences,
		InvalidatedInferences: em.InvalidatedInferences,
		CoinBalance:           em.CoinBalance,
		EpochsCompleted:       em.EpochsCompleted,
		MissRatePercent:       state.MissRate(em.InferenceCount, em.MissedRequests),
		TimeslotAssigned:      c.epochNode.TimeslotAssigned,
		PocWeights:            copyMap(c.epochNode.PocWeights),
		StartTime:             startTime,
		EndTime:               endTime,
		DurationSeconds:       duration,
	}
	if walletOK {
		earned := math.Round((wallet-c.st.WalletBalanceAtEpochStart)*10000) / 10000
		snap.EarnedGNK = &earned
	}
	if c.estimatedReward > 0 {
		est := c.estimatedReward
		snap.EstimatedGNK = &est
	}

	// On-chain performance summary
	perf, err := c.f.FetchEpochPerfSummary(c.cfg.NodeRESTURL, addr, epoch)
	if err != nil {
		slog.Warn("epoch performance summary", "epoch", epoch, "err", err)
	} else {
		snap.RewardedGNK = &perf.RewardedGNK
		snap.Claimed = &perf.Claimed
	}

	epochStr := fmt.Sprintf("%d", epoch)
	if c.history[addr] == nil {
		c.history[addr] = make(map[string]*state.EpochSnapshot)
	}
	c.history[addr][epochStr] = snap
	state.SaveHistory(c.cfg.HistoryFile, c.history, c.cfg.MaxHistory)

	ce := strconv.FormatInt(epoch, 10)
	c.m.EpochInferences.WithLabelValues(addr, ce).Set(float64(snap.InferenceCount))
	c.m.EpochMissed.WithLabelValues(addr, ce).Set(float64(snap.MissedRequests))
	c.m.EpochEarnedCoins.WithLabelValues(addr, ce).Set(float64(snap.EarnedCoins))
	c.m.EpochValidated.WithLabelValues(addr, ce).Set(float64(snap.ValidatedInferences))
	c.m.EpochInvalidated.WithLabelValues(addr, ce).Set(float64(snap.InvalidatedInferences))
	c.m.EpochCoinBalance.WithLabelValues(addr, ce).Set(float64(snap.CoinBalance))
	c.m.EpochDone.WithLabelValues(addr, ce).Set(float64(snap.EpochsCompleted))
	c.m.EpochMissRate.WithLabelValues(addr, ce).Set(snap.MissRatePercent)
	c.m.EpochTimeslot.WithLabelValues(addr, ce).Set(float64(snap.TimeslotAssigned))
	for nodeID, pw := range snap.PocWeights {
		c.m.EpochPocWeight.WithLabelValues(addr, ce, nodeID).Set(float64(pw))
	}
	c.m.EpochStartTime.WithLabelValues(addr, ce).Set(snap.StartTime)
	c.m.EpochEndTime.WithLabelValues(addr, ce).Set(snap.EndTime)
	c.m.EpochDuration.WithLabelValues(addr, ce).Set(snap.DurationSeconds)
	if snap.EarnedGNK != nil {
		c.m.EpochEarnedGNK.WithLabelValues(addr, ce).Set(*snap.EarnedGNK)
	}
	if snap.RewardedGNK != nil {
		c.m.EpochRewardedGNK.WithLabelValues(addr, ce).Set(*snap.RewardedGNK)
	}
	if snap.EstimatedGNK != nil {
		c.m.EpochEstimated.WithLabelValues(addr, ce).Set(*snap.EstimatedGNK)
	}
	if snap.Claimed != nil {
		c.m.EpochClaimed.WithLabelValues(addr, ce).Set(float64(*snap.Claimed))
	}

	slog.Info("epoch snapshot saved",
		"epoch", epoch,
		"inferences", snap.InferenceCount,
		"earned_coins", snap.EarnedCoins,
		"earned_gnk", snap.EarnedGNK,
		"rewarded_gnk", snap.RewardedGNK,
		"missed", snap.MissedRequests,
		"miss_rate", snap.MissRatePercent,
		"timeslot", snap.TimeslotAssigned,
		"duration_s", int(snap.DurationSeconds),
	)

	c.epochNode = state.NewEpochNodeState()
}

// --- Node metrics ---

func (c *Collector) collectNodes() {
	addr := c.cfg.Participant
	if addr == "" {
		addr = "unknown"
	}

	nodes, err := c.f.FetchNodes(c.cfg.AdminAPIURL)
	if err != nil {
		slog.Warn("fetch nodes", "err", err)
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, entry := range nodes {
		wg.Add(1)
		go func(entry fetcher.NodeEntry) {
			defer wg.Done()
			c.collectOneNode(addr, entry, &mu)
		}(entry)
	}
	wg.Wait()
}

// collectOneNode collects all metrics for a single node.
// mu protects writes to c.epochNode which is shared across parallel node goroutines.
func (c *Collector) collectOneNode(addr string, entry fetcher.NodeEntry, mu *sync.Mutex) {
	ni := entry.Node
	nodeID := ni.ID
	host := ni.Host
	st := entry.State

	for _, hw := range ni.Hardware {
		c.m.NodeHardwareInfo.WithLabelValues(addr, nodeID, host, hw.Type, strconv.Itoa(hw.Count)).Set(1)
	}

	c.m.NodeStatus.WithLabelValues(addr, nodeID, host).Set(nodeStatusVal(st.CurrentStatus))
	c.m.NodeIntended.WithLabelValues(addr, nodeID, host).Set(nodeStatusVal(st.IntendedStatus))
	c.m.PocCurrent.WithLabelValues(addr, nodeID, host).Set(pocStatusVal(st.PocCurrentStatus))
	c.m.PocIntended.WithLabelValues(addr, nodeID, host).Set(pocStatusVal(st.PocIntendedStatus))

	for model, md := range st.EpochMLNodes {
		if md.PocWeight != nil {
			pw := *md.PocWeight
			c.m.NodePocWeight.WithLabelValues(addr, nodeID, host, model).Set(float64(pw))

			mu.Lock()
			if pw > c.epochNode.PocWeights[nodeID] {
				c.epochNode.PocWeights[nodeID] = pw
			}
			curMax := c.epochNode.PocWeights[nodeID]
			mu.Unlock()

			if c.st.ChainEpoch > 0 {
				ce := strconv.FormatInt(c.st.ChainEpoch, 10)
				c.m.EpochPocWeight.WithLabelValues(addr, ce, nodeID).Set(float64(curMax))
			}
		}
		if len(md.TimeslotAllocation) > 0 && md.TimeslotAllocation[0] {
			c.m.NodeTimeslot.WithLabelValues(addr, nodeID, host, model).Set(1)
			mu.Lock()
			c.epochNode.TimeslotAssigned = 1
			mu.Unlock()
			if c.st.ChainEpoch > 0 {
				ce := strconv.FormatInt(c.st.ChainEpoch, 10)
				c.m.EpochTimeslot.WithLabelValues(addr, ce).Set(1)
			}
		} else {
			c.m.NodeTimeslot.WithLabelValues(addr, nodeID, host, model).Set(0)
		}
	}

	if ni.PocPort > 0 && host != "" {
		gpu := c.f.FetchGPUStats(host, ni.PocPort)
		c.m.NodeGPUCount.WithLabelValues(addr, nodeID, host).Set(float64(gpu.Count))
		c.m.NodeGPUUtil.WithLabelValues(addr, nodeID, host).Set(gpu.AvgUtil)
		for _, dev := range gpu.Devices {
			di := strconv.Itoa(dev.Index)
			c.m.NodeGPUDeviceUtil.WithLabelValues(addr, nodeID, host, di).Set(dev.UtilizationPercent)
			avail := 0.0
			if dev.IsAvailable {
				avail = 1.0
			}
			c.m.NodeGPUDeviceAvail.WithLabelValues(addr, nodeID, host, di).Set(avail)
			if dev.TemperatureC != nil {
				c.m.NodeGPUDeviceTemp.WithLabelValues(addr, nodeID, host, di).Set(*dev.TemperatureC)
			}
			if dev.TotalMemoryMB != nil {
				c.m.NodeGPUDeviceMemTotal.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.TotalMemoryMB))
			}
			if dev.FreeMemoryMB != nil {
				c.m.NodeGPUDeviceMemFree.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.FreeMemoryMB))
			}
			if dev.UsedMemoryMB != nil {
				c.m.NodeGPUDeviceMemUsed.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.UsedMemoryMB))
			}
		}

		// ML node service state
		if svcState, svcErr := c.f.FetchMLNodeState(host, ni.PocPort); svcErr != nil {
			slog.Debug("ml node state unavailable", "node", nodeID, "err", svcErr)
		} else {
			sv := mlNodeServiceStateMap[svcState]
			c.m.NodeServiceState.WithLabelValues(addr, nodeID, host).Set(sv)
		}

		// ML node disk space
		if diskGB, diskErr := c.f.FetchMLNodeDiskSpaceGB(host, ni.PocPort); diskErr != nil {
			slog.Debug("ml node disk space unavailable", "node", nodeID, "err", diskErr)
		} else {
			c.m.NodeDiskAvailableGB.WithLabelValues(addr, nodeID, host).Set(diskGB)
		}

		// GPU driver info
		if drvInfo, drvErr := c.f.FetchGPUDriverInfo(host, ni.PocPort); drvErr != nil {
			slog.Debug("gpu driver info unavailable", "node", nodeID, "err", drvErr)
		} else {
			c.m.NodeGPUDriverInfo.WithLabelValues(addr, nodeID, host, drvInfo.DriverVersion, drvInfo.CudaDriverVersion).Set(1)
		}

		// ML node manager health
		if health, healthErr := c.f.FetchMLNodeHealth(host, ni.PocPort); healthErr != nil {
			slog.Debug("ml node health unavailable", "node", nodeID, "err", healthErr)
		} else {
			c.setManagerMetric(addr, nodeID, host, "pow", health.ManagerPow)
			c.setManagerMetric(addr, nodeID, host, "inference", health.ManagerInference)
			c.setManagerMetric(addr, nodeID, host, "train", health.ManagerTrain)
		}
	}
}

func (c *Collector) setManagerMetric(addr, nodeID, host, manager string, s fetcher.MLNodeManagerStatus) {
	running := 0.0
	if s.Running {
		running = 1.0
	}
	healthy := 0.0
	if s.Healthy {
		healthy = 1.0
	}
	c.m.NodeManagerRunning.WithLabelValues(addr, nodeID, host, manager).Set(running)
	c.m.NodeManagerHealthy.WithLabelValues(addr, nodeID, host, manager).Set(healthy)
}

// --- Restore metrics from history on startup ---

func (c *Collector) restoreMetrics() {
	total := 0
	for participant, epochs := range c.history {
		for epochStr, snap := range epochs {
			p := participant
			e := epochStr
			c.m.EpochInferences.WithLabelValues(p, e).Set(float64(snap.InferenceCount))
			c.m.EpochMissed.WithLabelValues(p, e).Set(float64(snap.MissedRequests))
			c.m.EpochEarnedCoins.WithLabelValues(p, e).Set(float64(snap.EarnedCoins))
			c.m.EpochValidated.WithLabelValues(p, e).Set(float64(snap.ValidatedInferences))
			c.m.EpochInvalidated.WithLabelValues(p, e).Set(float64(snap.InvalidatedInferences))
			c.m.EpochCoinBalance.WithLabelValues(p, e).Set(float64(snap.CoinBalance))
			c.m.EpochDone.WithLabelValues(p, e).Set(float64(snap.EpochsCompleted))
			c.m.EpochMissRate.WithLabelValues(p, e).Set(snap.MissRatePercent)
			c.m.EpochTimeslot.WithLabelValues(p, e).Set(float64(snap.TimeslotAssigned))
			for nodeID, pw := range snap.PocWeights {
				c.m.EpochPocWeight.WithLabelValues(p, e, nodeID).Set(float64(pw))
			}
			if snap.StartTime > 0 {
				c.m.EpochStartTime.WithLabelValues(p, e).Set(snap.StartTime)
			}
			if snap.EndTime > 0 {
				c.m.EpochEndTime.WithLabelValues(p, e).Set(snap.EndTime)
			}
			if snap.DurationSeconds > 0 {
				c.m.EpochDuration.WithLabelValues(p, e).Set(snap.DurationSeconds)
			}
			if snap.EarnedGNK != nil {
				c.m.EpochEarnedGNK.WithLabelValues(p, e).Set(*snap.EarnedGNK)
			}
			if snap.RewardedGNK != nil {
				c.m.EpochRewardedGNK.WithLabelValues(p, e).Set(*snap.RewardedGNK)
			}
			if snap.EstimatedGNK != nil {
				c.m.EpochEstimated.WithLabelValues(p, e).Set(*snap.EstimatedGNK)
			}
			if snap.Claimed != nil {
				c.m.EpochClaimed.WithLabelValues(p, e).Set(float64(*snap.Claimed))
			}
			total++
		}
	}
	if total > 0 {
		slog.Info("restored epoch metrics from history", "participants", len(c.history), "total_epochs", total)
	}
}

func (c *Collector) pruneEpochMax() {
	keys := make([]int64, 0, len(c.epochMax))
	for k := range c.epochMax {
		keys = append(keys, k)
	}
	if len(keys) <= 5 {
		return
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys[:len(keys)-5] {
		delete(c.epochMax, k)
	}
}

func nodeStatusVal(s string) float64 {
	v, ok := nodeStatusMap[strings.ToUpper(s)]
	if !ok && s != "" {
		slog.Warn("unknown node status", "status", s)
	}
	return v
}

func pocStatusVal(s string) float64 {
	v, ok := pocStatusMap[strings.ToUpper(s)]
	if !ok && s != "" {
		slog.Warn("unknown poc status", "status", s)
	}
	return v
}

func copyMap(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// --- Stats ---

func (c *Collector) collectStats() {
	// Per-model stats
	models, err := c.f.FetchStatsModels(c.cfg.APIURL)
	if err != nil {
		slog.Debug("stats models unavailable", "err", err)
	} else {
		for _, m := range models {
			if m.Model == "" {
				continue
			}
			c.m.StatsModelAiTokens.WithLabelValues(m.Model).Set(float64(m.AiTokens))
			c.m.StatsModelInferences.WithLabelValues(m.Model).Set(float64(m.Inferences))
		}
	}

	// Network-wide summary
	summary, err := c.f.FetchStatsSummary(c.cfg.APIURL)
	if err != nil {
		slog.Debug("stats summary unavailable", "err", err)
		return
	}
	c.m.StatsAiTokens.Set(float64(summary.AiTokens))
	c.m.StatsInferences.Set(float64(summary.Inferences))
	c.m.StatsActualCost.Set(float64(summary.ActualCost))
}

// --- Bridge ---

func (c *Collector) collectBridge() {
	status, err := c.f.FetchBridgeStatus(c.cfg.APIURL)
	if err != nil {
		slog.Debug("bridge status unavailable", "err", err)
		return
	}
	c.m.BridgePendingBlocks.Set(float64(status.PendingBlocks))
	c.m.BridgePendingReceipts.Set(float64(status.PendingReceipts))
	ready := 0.0
	if status.ReadyToProcess {
		ready = 1.0
	}
	c.m.BridgeReadyToProcess.Set(ready)
	if status.EarliestBlock > 0 {
		c.m.BridgeEarliestBlock.Set(float64(status.EarliestBlock))
	}
	if status.LatestBlock > 0 {
		c.m.BridgeLatestBlock.Set(float64(status.LatestBlock))
	}
}

// --- Tokenomics ---

func (c *Collector) collectTokenomics() {
	tok, err := c.f.FetchTokenomics(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Debug("tokenomics unavailable", "err", err)
		return
	}
	c.m.TokenomicsTotalFees.Set(float64(tok.TotalFees))
	c.m.TokenomicsTotalSubsidies.Set(float64(tok.TotalSubsidies))
	c.m.TokenomicsTotalRefunded.Set(float64(tok.TotalRefunded))
	c.m.TokenomicsTotalBurned.Set(float64(tok.TotalBurned))
}

// --- PoC v2 ---

func (c *Collector) collectPoCv2() {
	addr := c.cfg.Participant
	if addr == "" || c.st.PocStartBlockHeight == 0 {
		return
	}

	// Artifact count
	commit, err := c.f.FetchPoCv2Commit(c.cfg.NodeRESTURL, c.st.PocStartBlockHeight, addr)
	if err != nil {
		slog.Debug("poc_v2 commit unavailable", "err", err)
	} else {
		c.m.PoCv2ArtifactCount.WithLabelValues(addr).Set(float64(commit.Count))
	}

	// Per-node weight distribution
	weights, err := c.f.FetchMLNodeWeightDist(c.cfg.NodeRESTURL, c.st.PocStartBlockHeight, addr)
	if err != nil {
		slog.Debug("poc_v2 node weight dist unavailable", "err", err)
		return
	}
	for _, w := range weights {
		if w.NodeID == "" {
			continue
		}
		c.m.PoCv2NodeWeight.WithLabelValues(addr, w.NodeID).Set(float64(w.Weight))
	}
}
