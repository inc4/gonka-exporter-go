package collector

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

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
	cfg              config.Config
	st               state.EpochState
	history          state.History
	epochMax         map[int64]*state.EpochMaxValues
	epochNode        state.EpochNodeState
	estimatedReward  float64 // last calculated estimated reward for current epoch
}

// New creates a Collector, restoring persisted state and history.
func New(cfg config.Config) *Collector {
	st      := state.LoadState(cfg.StateFile)
	history := state.LoadHistory(cfg.HistoryFile)
	c := &Collector{
		cfg:       cfg,
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
		h, t := fetcher.FetchMaxBlockHeightFromNodes(c.cfg.BlockHeightNodes)
		if h > 0 {
			metrics.BlockHeightMax.WithLabelValues(p).Set(float64(h))
		}
		if t != "" {
			ts := strings.TrimSuffix(t, "Z")
			if parsed, err := time.Parse("2006-01-02T15:04:05.999999999", ts); err == nil {
				metrics.BlockTime.WithLabelValues(p).Set(float64(parsed.Unix()))
			}
		}
	}()

	status, err := fetcher.FetchTendermintStatus(c.cfg.NodeRPCURL)
	if err != nil {
		slog.Warn("tendermint status", "err", err)
		return
	}
	si := status.Result.SyncInfo
	if h, err := strconv.ParseInt(si.LatestBlockHeight, 10, 64); err == nil {
		metrics.BlockHeight.WithLabelValues(p).Set(float64(h))
	}
	if si.LatestBlockTime != "" {
		ts := strings.TrimSuffix(si.LatestBlockTime, "Z")
		if parsed, err := time.Parse("2006-01-02T15:04:05.999999999", ts); err == nil {
			metrics.BlockTime.WithLabelValues(p).Set(float64(parsed.Unix()))
		}
	}
	catching := 0.0
	if si.CatchingUp {
		catching = 1.0
	}
	metrics.CatchingUp.WithLabelValues(p).Set(catching)
}

// --- Network-wide participants ---

func (c *Collector) collectNetworkParticipants() {
	participants, err := fetcher.FetchNetworkParticipants(c.cfg.APIURL)
	if err != nil {
		slog.Warn("network participants", "err", err)
		return
	}
	metrics.NetTotalParticipantCount.Set(float64(len(participants)))
	active := 0
	for _, p := range participants {
		addr := p.Seed.Participant
		if addr == "" {
			continue
		}
		active++
		if p.Weight != nil {
			metrics.NetParticipantWeight.WithLabelValues(addr).Set(*p.Weight)
		}
		for _, group := range p.MLNodes {
			for _, node := range group.MLNodes {
				if node.NodeID != "" && node.PocWeight != nil {
					metrics.NetNodePocWeight.WithLabelValues(addr, node.NodeID).Set(*node.PocWeight)
				}
			}
		}
	}
	metrics.NetActiveParticipantCount.Set(float64(active))
}

// --- Pricing and models ---

func (c *Collector) collectPricingAndModels() {
	pricing, err := fetcher.FetchPricing(c.cfg.APIURL)
	if err != nil {
		slog.Warn("pricing", "err", err)
	} else {
		if pricing.UnitOfComputePrice != nil {
			metrics.PricingUoC.Set(*pricing.UnitOfComputePrice)
		}
		if pricing.DynamicPricingEnabled != nil {
			v := 0.0
			if *pricing.DynamicPricingEnabled {
				v = 1.0
			}
			metrics.PricingDynamic.Set(v)
		}
		for _, m := range pricing.Models {
			if m.ID == "" {
				continue
			}
			if m.PricePerToken != nil {
				metrics.ModelPrice.WithLabelValues(m.ID).Set(*m.PricePerToken)
			}
			if m.UnitsOfComputePerToken != nil {
				metrics.ModelUnits.WithLabelValues(m.ID).Set(*m.UnitsOfComputePerToken)
			}
			if m.Utilization != nil {
				metrics.ModelUtilization.WithLabelValues(m.ID).Set(*m.Utilization * 100)
			}
			if m.Capacity != nil {
				metrics.ModelCapacity.WithLabelValues(m.ID).Set(float64(*m.Capacity))
			}
		}
	}

	models, err := fetcher.FetchModels(c.cfg.APIURL)
	if err != nil {
		slog.Warn("models", "err", err)
		return
	}
	for _, m := range models.Models {
		if m.ID == "" {
			continue
		}
		if m.VRAM != nil {
			metrics.ModelVRAM.WithLabelValues(m.ID).Set(*m.VRAM)
		}
		if m.ThroughputPerNonce != nil {
			metrics.ModelThroughput.WithLabelValues(m.ID).Set(*m.ThroughputPerNonce)
		}
		if m.ValidationThreshold != nil {
			metrics.ModelValThresh.WithLabelValues(m.ID).Set(
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

	// 1. Chain epoch
	chainEpoch, err := fetcher.FetchCurrentEpoch(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch current epoch", "err", err)
	} else {
		metrics.ChainEpoch.WithLabelValues(addr).Set(float64(chainEpoch))
	}

	// 2. Save prev start time before updating
	prevEpochStartTime := c.st.EpochStartTime

	// 3. Epoch info — real start time from blockchain
	var epochInfo *fetcher.EpochInfo
	epochInfo, err = fetcher.FetchEpochInfo(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch epoch info", "err", err)
	} else {
		c.st.EpochLength = epochInfo.EpochLength
		if epochInfo.PocStartBlockHeight != c.st.PocStartBlockHeight {
			if t, err := fetcher.FetchBlockTimeAtHeight(c.cfg.NodeRPCURL, epochInfo.PocStartBlockHeight); err != nil {
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
	}

	// 4. Epoch group data — total network weight + estimated reward + reputation + network inference count
	groupData, err := fetcher.FetchEpochGroupData(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Warn("fetch epoch group data", "err", err)
	} else {
		if groupData.NumberOfRequests > 0 {
			metrics.NetEpochInferenceCount.WithLabelValues(addr).Set(float64(groupData.NumberOfRequests))
		}
		if groupData.TotalWeight > 0 {
			// emission(N) = 323000 × exp(-0.000475 × (N-1))
			emission := 323000.0 * math.Exp(-0.000475*float64(groupData.EpochIndex-1))
			rewardPerWeight := emission / float64(groupData.TotalWeight)
			metrics.NetTotalWeight.WithLabelValues(addr).Set(float64(groupData.TotalWeight))
			metrics.NetRewardPerWeight.WithLabelValues(addr).Set(rewardPerWeight)
			for _, vw := range groupData.ValidationWeights {
				if vw.MemberAddress == addr {
					myWeight, _ := strconv.ParseInt(vw.Weight, 10, 64)
					c.estimatedReward = float64(myWeight) * rewardPerWeight
					metrics.ParticipantReputation.WithLabelValues(addr).Set(float64(vw.Reputation))
					if chainEpoch > 0 {
						ce := strconv.FormatInt(chainEpoch, 10)
						metrics.EpochEstimated.WithLabelValues(addr, ce).Set(c.estimatedReward)
					}
					break
				}
			}
		}
	}

	// 4b. BLS DKG phase
	if chainEpoch > 0 {
		bls, blsErr := fetcher.FetchBLSEpoch(c.cfg.APIURL, chainEpoch)
		if blsErr != nil {
			slog.Warn("fetch bls epoch", "err", blsErr)
		} else {
			metrics.BLSDKGPhase.WithLabelValues(addr).Set(float64(bls.DKGPhase))
			if bls.DealingPhaseDeadlineBlock > 0 {
				metrics.BLSDealingDeadline.WithLabelValues(addr).Set(float64(bls.DealingPhaseDeadlineBlock))
			}
			if bls.VerifyingPhaseDeadlineBlock > 0 {
				metrics.BLSVerifyingDeadline.WithLabelValues(addr).Set(float64(bls.VerifyingPhaseDeadlineBlock))
			}
		}
	}

	// 5. Participant stats
	stats, err := fetcher.FetchParticipantStats(c.cfg.NodeRESTURL, addr)
	if err != nil {
		slog.Warn("fetch participant stats", "err", err)
		return
	}
	metrics.ParticipantEpochsDone.WithLabelValues(addr).Set(float64(stats.EpochsCompleted))
	metrics.ParticipantCoinBalance.WithLabelValues(addr).Set(float64(stats.CoinBalance))
	metrics.ParticipantInferences.WithLabelValues(addr).Set(float64(stats.InferenceCount))
	metrics.ParticipantMissed.WithLabelValues(addr).Set(float64(stats.MissedRequests))
	metrics.ParticipantEarnedCoins.WithLabelValues(addr).Set(float64(stats.EarnedCoins))
	metrics.ParticipantValidated.WithLabelValues(addr).Set(float64(stats.ValidatedInferences))
	metrics.ParticipantInvalidated.WithLabelValues(addr).Set(float64(stats.InvalidatedInferences))
	// Extended participant health
	if sv, ok := participantStatusMap[stats.Status]; ok {
		metrics.ParticipantStatus.WithLabelValues(addr).Set(sv)
	}
	metrics.ParticipantConsecutiveInv.WithLabelValues(addr).Set(float64(stats.ConsecutiveInvalidInferences))
	metrics.ParticipantBurnedCoins.WithLabelValues(addr).Set(float64(stats.BurnedCoins))
	metrics.ParticipantRewardedCoins.WithLabelValues(addr).Set(float64(stats.RewardedCoins))

	// 6. Wallet balance
	wallet, walletErr := fetcher.FetchWalletBalance(c.cfg.NodeRESTURL, addr)
	if walletErr != nil {
		slog.Warn("fetch wallet balance", "err", walletErr)
	} else {
		metrics.ParticipantWallet.WithLabelValues(addr).Set(wallet)
		if c.st.WalletBalanceAtEpochStart == 0 {
			c.st.WalletBalanceAtEpochStart = wallet
		}
	}

	// 7. Per-epoch max tracking + live gauges
	if chainEpoch > 0 {
		if _, ok := c.epochMax[chainEpoch]; !ok {
			c.epochMax[chainEpoch] = &state.EpochMaxValues{}
		}
		em := c.epochMax[chainEpoch]
		em.InferenceCount        = max64(em.InferenceCount,        stats.InferenceCount)
		em.MissedRequests        = max64(em.MissedRequests,        stats.MissedRequests)
		em.EarnedCoins           = max64(em.EarnedCoins,           stats.EarnedCoins)
		em.ValidatedInferences   = max64(em.ValidatedInferences,   stats.ValidatedInferences)
		em.InvalidatedInferences = max64(em.InvalidatedInferences, stats.InvalidatedInferences)
		em.CoinBalance           = max64(em.CoinBalance,           stats.CoinBalance)
		em.EpochsCompleted       = max64(em.EpochsCompleted,       stats.EpochsCompleted)

		ce := strconv.FormatInt(chainEpoch, 10)
		metrics.EpochInferences.WithLabelValues(addr, ce).Set(float64(em.InferenceCount))
		metrics.EpochMissed.WithLabelValues(addr, ce).Set(float64(em.MissedRequests))
		metrics.EpochEarnedCoins.WithLabelValues(addr, ce).Set(float64(em.EarnedCoins))
		metrics.EpochValidated.WithLabelValues(addr, ce).Set(float64(em.ValidatedInferences))
		metrics.EpochInvalidated.WithLabelValues(addr, ce).Set(float64(em.InvalidatedInferences))
		metrics.EpochCoinBalance.WithLabelValues(addr, ce).Set(float64(em.CoinBalance))
		metrics.EpochDone.WithLabelValues(addr, ce).Set(float64(em.EpochsCompleted))
		metrics.EpochMissRate.WithLabelValues(addr, ce).Set(state.MissRate(em.InferenceCount, em.MissedRequests))

		if walletErr == nil {
			metrics.EpochEarnedGNK.WithLabelValues(addr, ce).Set(wallet - c.st.WalletBalanceAtEpochStart)
		}
		if c.st.EpochStartTime > 0 {
			metrics.EpochStartTime.WithLabelValues(addr, ce).Set(c.st.EpochStartTime)
		}
		if epochInfo != nil && c.st.EpochStartTime > 0 && c.st.PocStartBlockHeight > 0 && c.st.EpochLength > 0 {
			blocksElapsed := epochInfo.BlockHeight - c.st.PocStartBlockHeight
			timeElapsed   := now - c.st.EpochStartTime
			if blocksElapsed > 0 && timeElapsed > 0 {
				avgBlockTime    := timeElapsed / float64(blocksElapsed)
				blocksRemaining := c.st.EpochLength - blocksElapsed
				metrics.EpochEndTime.WithLabelValues(addr, ce).Set(now + float64(blocksRemaining)*avgBlockTime)
			}
		}
	}

	// 8. Epoch boundary detection
	prevEpoch := c.st.ChainEpoch
	if c.st.Valid && prevEpoch > 0 && chainEpoch > prevEpoch {
		if chainEpoch > prevEpoch+1 {
			slog.Warn("skipped epochs", "from", prevEpoch, "to", chainEpoch)
		}
		c.recordSnapshot(prevEpoch, prevEpochStartTime, c.st.EpochStartTime, wallet, walletErr == nil)
		c.st.WalletBalanceAtEpochStart = wallet
		c.pruneEpochMax()
	} else if !c.st.Valid {
		c.st.WalletBalanceAtEpochStart = wallet
	}

	// 9. Persist state on epoch change
	changed := c.st.ChainEpoch != chainEpoch
	c.st.ChainEpoch = chainEpoch
	c.st.Valid = true
	if changed {
		state.SaveState(c.cfg.StateFile, c.st)
	}
}

func (c *Collector) recordSnapshot(epoch int64, startTime, endTime float64, wallet float64, walletOK bool) {
	addr := c.cfg.Participant
	em   := c.epochMax[epoch]
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
	perf, err := fetcher.FetchEpochPerfSummary(c.cfg.NodeRESTURL, addr, epoch)
	if err != nil {
		slog.Warn("epoch performance summary", "epoch", epoch, "err", err)
	} else {
		snap.RewardedGNK = &perf.RewardedGNK
		snap.Claimed     = &perf.Claimed
	}

	epochStr := fmt.Sprintf("%d", epoch)
	if c.history[addr] == nil {
		c.history[addr] = make(map[string]*state.EpochSnapshot)
	}
	c.history[addr][epochStr] = snap
	state.SaveHistory(c.cfg.HistoryFile, c.history, c.cfg.MaxHistory)

	ce := strconv.FormatInt(epoch, 10)
	metrics.EpochInferences.WithLabelValues(addr, ce).Set(float64(snap.InferenceCount))
	metrics.EpochMissed.WithLabelValues(addr, ce).Set(float64(snap.MissedRequests))
	metrics.EpochEarnedCoins.WithLabelValues(addr, ce).Set(float64(snap.EarnedCoins))
	metrics.EpochValidated.WithLabelValues(addr, ce).Set(float64(snap.ValidatedInferences))
	metrics.EpochInvalidated.WithLabelValues(addr, ce).Set(float64(snap.InvalidatedInferences))
	metrics.EpochCoinBalance.WithLabelValues(addr, ce).Set(float64(snap.CoinBalance))
	metrics.EpochDone.WithLabelValues(addr, ce).Set(float64(snap.EpochsCompleted))
	metrics.EpochMissRate.WithLabelValues(addr, ce).Set(snap.MissRatePercent)
	metrics.EpochTimeslot.WithLabelValues(addr, ce).Set(float64(snap.TimeslotAssigned))
	for nodeID, pw := range snap.PocWeights {
		metrics.EpochPocWeight.WithLabelValues(addr, ce, nodeID).Set(float64(pw))
	}
	metrics.EpochStartTime.WithLabelValues(addr, ce).Set(snap.StartTime)
	metrics.EpochEndTime.WithLabelValues(addr, ce).Set(snap.EndTime)
	metrics.EpochDuration.WithLabelValues(addr, ce).Set(snap.DurationSeconds)
	if snap.EarnedGNK != nil {
		metrics.EpochEarnedGNK.WithLabelValues(addr, ce).Set(*snap.EarnedGNK)
	}
	if snap.RewardedGNK != nil {
		metrics.EpochRewardedGNK.WithLabelValues(addr, ce).Set(*snap.RewardedGNK)
	}
	if snap.EstimatedGNK != nil {
		metrics.EpochEstimated.WithLabelValues(addr, ce).Set(*snap.EstimatedGNK)
	}
	if snap.Claimed != nil {
		metrics.EpochClaimed.WithLabelValues(addr, ce).Set(float64(*snap.Claimed))
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

	nodes, err := fetcher.FetchNodes(c.cfg.AdminAPIURL)
	if err != nil {
		slog.Warn("fetch nodes", "err", err)
		return
	}

	for _, entry := range nodes {
		ni     := entry.Node
		nodeID := ni.ID
		host   := ni.Host
		st     := entry.State

		for _, hw := range ni.Hardware {
			metrics.NodeHardwareInfo.WithLabelValues(addr, nodeID, host, hw.Type, strconv.Itoa(hw.Count)).Set(1)
		}

		metrics.NodeStatus.WithLabelValues(addr, nodeID, host).Set(nodeStatusVal(st.CurrentStatus))
		metrics.NodeIntended.WithLabelValues(addr, nodeID, host).Set(nodeStatusVal(st.IntendedStatus))
		metrics.PocCurrent.WithLabelValues(addr, nodeID, host).Set(pocStatusVal(st.PocCurrentStatus))
		metrics.PocIntended.WithLabelValues(addr, nodeID, host).Set(pocStatusVal(st.PocIntendedStatus))

		for model, md := range st.EpochMLNodes {
			if md.PocWeight != nil {
				pw := *md.PocWeight
				metrics.NodePocWeight.WithLabelValues(addr, nodeID, host, model).Set(float64(pw))
				if pw > c.epochNode.PocWeights[nodeID] {
					c.epochNode.PocWeights[nodeID] = pw
				}
				if c.st.ChainEpoch > 0 {
					ce := strconv.FormatInt(c.st.ChainEpoch, 10)
					metrics.EpochPocWeight.WithLabelValues(addr, ce, nodeID).Set(float64(c.epochNode.PocWeights[nodeID]))
				}
			}
			if len(md.TimeslotAllocation) > 0 && md.TimeslotAllocation[0] {
				metrics.NodeTimeslot.WithLabelValues(addr, nodeID, host, model).Set(1)
				c.epochNode.TimeslotAssigned = 1
				if c.st.ChainEpoch > 0 {
					ce := strconv.FormatInt(c.st.ChainEpoch, 10)
					metrics.EpochTimeslot.WithLabelValues(addr, ce).Set(1)
				}
			} else {
				metrics.NodeTimeslot.WithLabelValues(addr, nodeID, host, model).Set(0)
			}
		}

		if ni.PocPort > 0 && host != "" {
			gpu := fetcher.FetchGPUStats(host, ni.PocPort)
			metrics.NodeGPUCount.WithLabelValues(addr, nodeID, host).Set(float64(gpu.Count))
			metrics.NodeGPUUtil.WithLabelValues(addr, nodeID, host).Set(gpu.AvgUtil)
			for _, dev := range gpu.Devices {
				di := strconv.Itoa(dev.Index)
				metrics.NodeGPUDeviceUtil.WithLabelValues(addr, nodeID, host, di).Set(dev.UtilizationPercent)
				avail := 0.0
				if dev.IsAvailable {
					avail = 1.0
				}
				metrics.NodeGPUDeviceAvail.WithLabelValues(addr, nodeID, host, di).Set(avail)
				if dev.TemperatureC != nil {
					metrics.NodeGPUDeviceTemp.WithLabelValues(addr, nodeID, host, di).Set(*dev.TemperatureC)
				}
				if dev.TotalMemoryMB != nil {
					metrics.NodeGPUDeviceMemTotal.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.TotalMemoryMB))
				}
				if dev.FreeMemoryMB != nil {
					metrics.NodeGPUDeviceMemFree.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.FreeMemoryMB))
				}
				if dev.UsedMemoryMB != nil {
					metrics.NodeGPUDeviceMemUsed.WithLabelValues(addr, nodeID, host, di).Set(float64(*dev.UsedMemoryMB))
				}
			}

			// ML node service state
			if svcState, svcErr := fetcher.FetchMLNodeState(host, ni.PocPort); svcErr != nil {
				slog.Debug("ml node state unavailable", "node", nodeID, "err", svcErr)
			} else {
				sv := mlNodeServiceStateMap[svcState]
				metrics.NodeServiceState.WithLabelValues(addr, nodeID, host).Set(sv)
			}

			// ML node disk space
			if diskGB, diskErr := fetcher.FetchMLNodeDiskSpaceGB(host, ni.PocPort); diskErr != nil {
				slog.Debug("ml node disk space unavailable", "node", nodeID, "err", diskErr)
			} else {
				metrics.NodeDiskAvailableGB.WithLabelValues(addr, nodeID, host).Set(diskGB)
			}

			// GPU driver info
			if drvInfo, drvErr := fetcher.FetchGPUDriverInfo(host, ni.PocPort); drvErr != nil {
				slog.Debug("gpu driver info unavailable", "node", nodeID, "err", drvErr)
			} else {
				metrics.NodeGPUDriverInfo.WithLabelValues(addr, nodeID, host, drvInfo.DriverVersion, drvInfo.CudaDriverVersion).Set(1)
			}

			// ML node manager health
			if health, healthErr := fetcher.FetchMLNodeHealth(host, ni.PocPort); healthErr != nil {
				slog.Debug("ml node health unavailable", "node", nodeID, "err", healthErr)
			} else {
				setManagerMetric(addr, nodeID, host, "pow", health.ManagerPow)
				setManagerMetric(addr, nodeID, host, "inference", health.ManagerInference)
				setManagerMetric(addr, nodeID, host, "train", health.ManagerTrain)
			}
		}
	}
}

func setManagerMetric(addr, nodeID, host, manager string, s fetcher.MLNodeManagerStatus) {
	running := 0.0
	if s.Running {
		running = 1.0
	}
	healthy := 0.0
	if s.Healthy {
		healthy = 1.0
	}
	metrics.NodeManagerRunning.WithLabelValues(addr, nodeID, host, manager).Set(running)
	metrics.NodeManagerHealthy.WithLabelValues(addr, nodeID, host, manager).Set(healthy)
}

// --- Restore metrics from history on startup ---

func (c *Collector) restoreMetrics() {
	total := 0
	for participant, epochs := range c.history {
		for epochStr, snap := range epochs {
			p := participant
			e := epochStr
			metrics.EpochInferences.WithLabelValues(p, e).Set(float64(snap.InferenceCount))
			metrics.EpochMissed.WithLabelValues(p, e).Set(float64(snap.MissedRequests))
			metrics.EpochEarnedCoins.WithLabelValues(p, e).Set(float64(snap.EarnedCoins))
			metrics.EpochValidated.WithLabelValues(p, e).Set(float64(snap.ValidatedInferences))
			metrics.EpochInvalidated.WithLabelValues(p, e).Set(float64(snap.InvalidatedInferences))
			metrics.EpochCoinBalance.WithLabelValues(p, e).Set(float64(snap.CoinBalance))
			metrics.EpochDone.WithLabelValues(p, e).Set(float64(snap.EpochsCompleted))
			metrics.EpochMissRate.WithLabelValues(p, e).Set(snap.MissRatePercent)
			metrics.EpochTimeslot.WithLabelValues(p, e).Set(float64(snap.TimeslotAssigned))
			for nodeID, pw := range snap.PocWeights {
				metrics.EpochPocWeight.WithLabelValues(p, e, nodeID).Set(float64(pw))
			}
			if snap.StartTime > 0 {
				metrics.EpochStartTime.WithLabelValues(p, e).Set(snap.StartTime)
			}
			if snap.EndTime > 0 {
				metrics.EpochEndTime.WithLabelValues(p, e).Set(snap.EndTime)
			}
			if snap.DurationSeconds > 0 {
				metrics.EpochDuration.WithLabelValues(p, e).Set(snap.DurationSeconds)
			}
			if snap.EarnedGNK != nil {
				metrics.EpochEarnedGNK.WithLabelValues(p, e).Set(*snap.EarnedGNK)
			}
			if snap.RewardedGNK != nil {
				metrics.EpochRewardedGNK.WithLabelValues(p, e).Set(*snap.RewardedGNK)
			}
			if snap.EstimatedGNK != nil {
				metrics.EpochEstimated.WithLabelValues(p, e).Set(*snap.EstimatedGNK)
			}
			if snap.Claimed != nil {
				metrics.EpochClaimed.WithLabelValues(p, e).Set(float64(*snap.Claimed))
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
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
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

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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
	models, err := fetcher.FetchStatsModels(c.cfg.APIURL)
	if err != nil {
		slog.Debug("stats models unavailable", "err", err)
	} else {
		for _, m := range models {
			if m.Model == "" {
				continue
			}
			metrics.StatsModelAiTokens.WithLabelValues(m.Model).Set(float64(m.AiTokens))
			metrics.StatsModelInferences.WithLabelValues(m.Model).Set(float64(m.Inferences))
		}
	}

	// Network-wide summary
	summary, err := fetcher.FetchStatsSummary(c.cfg.APIURL)
	if err != nil {
		slog.Debug("stats summary unavailable", "err", err)
		return
	}
	metrics.StatsAiTokens.Set(float64(summary.AiTokens))
	metrics.StatsInferences.Set(float64(summary.Inferences))
	metrics.StatsActualCost.Set(float64(summary.ActualCost))
}

// --- Bridge ---

func (c *Collector) collectBridge() {
	status, err := fetcher.FetchBridgeStatus(c.cfg.APIURL)
	if err != nil {
		slog.Debug("bridge status unavailable", "err", err)
		return
	}
	metrics.BridgePendingBlocks.Set(float64(status.PendingBlocks))
	metrics.BridgePendingReceipts.Set(float64(status.PendingReceipts))
	ready := 0.0
	if status.ReadyToProcess {
		ready = 1.0
	}
	metrics.BridgeReadyToProcess.Set(ready)
	if status.EarliestBlock > 0 {
		metrics.BridgeEarliestBlock.Set(float64(status.EarliestBlock))
	}
	if status.LatestBlock > 0 {
		metrics.BridgeLatestBlock.Set(float64(status.LatestBlock))
	}
}

// --- Tokenomics ---

func (c *Collector) collectTokenomics() {
	tok, err := fetcher.FetchTokenomics(c.cfg.NodeRESTURL)
	if err != nil {
		slog.Debug("tokenomics unavailable", "err", err)
		return
	}
	metrics.TokenomicsTotalFees.Set(float64(tok.TotalFees))
	metrics.TokenomicsTotalSubsidies.Set(float64(tok.TotalSubsidies))
	metrics.TokenomicsTotalRefunded.Set(float64(tok.TotalRefunded))
	metrics.TokenomicsTotalBurned.Set(float64(tok.TotalBurned))
	metrics.TokenomicsTopRewardStart.Set(float64(tok.TopRewardStart))
}

// --- PoC v2 ---

func (c *Collector) collectPoCv2() {
	addr := c.cfg.Participant
	if addr == "" || c.st.PocStartBlockHeight == 0 {
		return
	}

	// Artifact count
	commit, err := fetcher.FetchPoCv2Commit(c.cfg.NodeRESTURL, c.st.PocStartBlockHeight, addr)
	if err != nil {
		slog.Debug("poc_v2 commit unavailable", "err", err)
	} else {
		metrics.PoCv2ArtifactCount.WithLabelValues(addr).Set(float64(commit.Count))
	}

	// Per-node weight distribution
	weights, err := fetcher.FetchMLNodeWeightDist(c.cfg.NodeRESTURL, c.st.PocStartBlockHeight, addr)
	if err != nil {
		slog.Debug("poc_v2 node weight dist unavailable", "err", err)
		return
	}
	for _, w := range weights {
		if w.NodeID == "" {
			continue
		}
		metrics.PoCv2NodeWeight.WithLabelValues(addr, w.NodeID).Set(float64(w.Weight))
	}
}
