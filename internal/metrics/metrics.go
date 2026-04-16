package metrics

import "github.com/prometheus/client_golang/prometheus"

// Metrics holds all Prometheus gauge definitions for the exporter.
// Create with NewMetrics(reg) — no global registration.
type Metrics struct {
	// Chain / sync
	BlockHeight        *prometheus.GaugeVec
	BlockHeightMax     *prometheus.GaugeVec
	BlockTimeLocal     *prometheus.GaugeVec // timestamp from local node (/status)
	BlockTimeNetwork   *prometheus.GaugeVec // timestamp from public network nodes
	CatchingUp         *prometheus.GaugeVec
	ChainEpoch         *prometheus.GaugeVec

	// Node hardware / status
	NodeStatus       *prometheus.GaugeVec
	NodeIntended     *prometheus.GaugeVec
	PocCurrent       *prometheus.GaugeVec
	PocIntended      *prometheus.GaugeVec
	NodePocWeight    *prometheus.GaugeVec
	NodeTimeslot     *prometheus.GaugeVec
	NodeGPUCount     *prometheus.GaugeVec
	NodeGPUUtil      *prometheus.GaugeVec
	NodeHardwareInfo *prometheus.GaugeVec

	// Node GPU — per device
	NodeGPUDeviceUtil     *prometheus.GaugeVec
	NodeGPUDeviceTemp     *prometheus.GaugeVec
	NodeGPUDeviceMemTotal *prometheus.GaugeVec
	NodeGPUDeviceMemFree  *prometheus.GaugeVec
	NodeGPUDeviceMemUsed  *prometheus.GaugeVec
	NodeGPUDeviceAvail    *prometheus.GaugeVec

	// Node ML service state
	NodeServiceState    *prometheus.GaugeVec
	NodeDiskAvailableGB *prometheus.GaugeVec

	// Network-wide
	NetParticipantWeight *prometheus.GaugeVec
	NetNodePocWeight     *prometheus.GaugeVec
	NetTotalWeight       *prometheus.GaugeVec
	NetRewardPerWeight   *prometheus.GaugeVec

	// Pricing / models
	PricingUoC      prometheus.Gauge
	PricingDynamic  prometheus.Gauge
	ModelPrice      *prometheus.GaugeVec
	ModelUnits      *prometheus.GaugeVec
	ModelVRAM       *prometheus.GaugeVec
	ModelThroughput *prometheus.GaugeVec
	ModelValThresh  *prometheus.GaugeVec

	// Participant live (current epoch)
	ParticipantEpochsDone  *prometheus.GaugeVec
	ParticipantCoinBalance *prometheus.GaugeVec
	ParticipantWallet      *prometheus.GaugeVec
	ParticipantInferences  *prometheus.GaugeVec
	ParticipantMissed      *prometheus.GaugeVec
	ParticipantEarnedCoins *prometheus.GaugeVec
	ParticipantValidated   *prometheus.GaugeVec
	ParticipantInvalidated *prometheus.GaugeVec

	// Participant health (extended)
	ParticipantStatus         *prometheus.GaugeVec
	ParticipantConsecutiveInv *prometheus.GaugeVec
	ParticipantBurnedCoins    *prometheus.GaugeVec
	ParticipantRewardedCoins  *prometheus.GaugeVec
	ParticipantReputation     *prometheus.GaugeVec

	// Network — counts and epoch-level
	NetActiveParticipantCount prometheus.Gauge
	NetTotalParticipantCount  prometheus.Gauge
	NetEpochInferenceCount    *prometheus.GaugeVec

	// BLS DKG phase
	BLSDKGPhase          *prometheus.GaugeVec
	BLSDealingDeadline   *prometheus.GaugeVec
	BLSVerifyingDeadline *prometheus.GaugeVec

	// Model utilization / capacity
	ModelUtilization *prometheus.GaugeVec
	ModelCapacity    *prometheus.GaugeVec

	// Epoch history (label epoch = chain epoch number as string)
	EpochInferences  *prometheus.GaugeVec
	EpochMissed      *prometheus.GaugeVec
	EpochEarnedCoins *prometheus.GaugeVec
	EpochValidated   *prometheus.GaugeVec
	EpochInvalidated *prometheus.GaugeVec
	EpochCoinBalance *prometheus.GaugeVec
	EpochDone        *prometheus.GaugeVec
	EpochMissRate    *prometheus.GaugeVec
	EpochPocWeight   *prometheus.GaugeVec
	EpochTimeslot    *prometheus.GaugeVec
	EpochStartTime   *prometheus.GaugeVec
	EpochEndTime     *prometheus.GaugeVec
	EpochDuration    *prometheus.GaugeVec
	EpochEarnedGNK   *prometheus.GaugeVec
	EpochRewardedGNK *prometheus.GaugeVec
	EpochClaimed     *prometheus.GaugeVec
	EpochEstimated   *prometheus.GaugeVec

	// Stats — network-wide inference statistics
	StatsAiTokens        prometheus.Gauge
	StatsInferences      prometheus.Gauge
	StatsActualCost      prometheus.Gauge
	StatsModelAiTokens   *prometheus.GaugeVec
	StatsModelInferences *prometheus.GaugeVec

	// Bridge — queue status for the Gonka bridge
	BridgePendingBlocks   prometheus.Gauge
	BridgePendingReceipts prometheus.Gauge
	BridgeReadyToProcess  prometheus.Gauge
	BridgeEarliestBlock   prometheus.Gauge
	BridgeLatestBlock     prometheus.Gauge

	// Node managers — running/healthy state per ML node manager
	NodeManagerRunning *prometheus.GaugeVec
	NodeManagerHealthy *prometheus.GaugeVec

	// GPU driver info — version in labels, value always 1
	NodeGPUDriverInfo *prometheus.GaugeVec

	// Tokenomics — chain-wide token flow counters
	TokenomicsTotalFees      prometheus.Gauge
	TokenomicsTotalSubsidies prometheus.Gauge
	TokenomicsTotalRefunded  prometheus.Gauge
	TokenomicsTotalBurned    prometheus.Gauge

	// PoC v2 — proof-of-compute artifact and weight data
	PoCv2ArtifactCount *prometheus.GaugeVec
	PoCv2NodeWeight    *prometheus.GaugeVec
}

// NewMetrics creates all Prometheus metrics and registers them with reg.
// Pass prometheus.DefaultRegisterer in production, prometheus.NewRegistry() in tests.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		// Chain / sync
		BlockHeight:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_block_height", Help: "Latest block height from local node"}, []string{"participant"}),
		BlockHeightMax: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_block_height_max", Help: "Maximum block height seen across public nodes"}, []string{"participant"}),
		BlockTimeLocal:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_block_time_local_seconds", Help: "Timestamp of latest block from local node (unix)"}, []string{"participant"}),
		BlockTimeNetwork: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_block_time_network_seconds", Help: "Timestamp of latest block from public network nodes (unix)"}, []string{"participant"}),
		CatchingUp:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_chain_catching_up", Help: "1 = syncing, 0 = synced"}, []string{"participant"}),
		ChainEpoch:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_chain_epoch", Help: "Current global chain epoch number"}, []string{"participant"}),

		// Node hardware / status
		NodeStatus:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_status", Help: "Node hardware status (0=UNKNOWN 1=INFERENCE 2=POC 3=TRAINING 4=STOPPED 5=FAILED)"}, []string{"participant", "node_id", "host"}),
		NodeIntended:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_intended_status", Help: "Node intended status"}, []string{"participant", "node_id", "host"}),
		PocCurrent:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_poc_current_status", Help: "PoC current status (0=IDLE 1=GENERATING 2=VALIDATING)"}, []string{"participant", "node_id", "host"}),
		PocIntended:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_poc_intended_status", Help: "PoC intended status"}, []string{"participant", "node_id", "host"}),
		NodePocWeight:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_poc_weight", Help: "PoC weight per node per model"}, []string{"participant", "node_id", "host", "model"}),
		NodeTimeslot:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_poc_timeslot_assigned", Help: "Timeslot assigned (1/0)"}, []string{"participant", "node_id", "host", "model"}),
		NodeGPUCount:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_count", Help: "GPU device count"}, []string{"participant", "node_id", "host"}),
		NodeGPUUtil:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_avg_utilization_percent", Help: "Average GPU utilization %"}, []string{"participant", "node_id", "host"}),
		NodeHardwareInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_hardware_info", Help: "Hardware info (value=1, metadata in labels)"}, []string{"participant", "node_id", "host", "hardware_type", "hardware_count"}),

		// Node GPU — per device
		NodeGPUDeviceUtil:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_utilization_percent", Help: "Per-device GPU compute utilization %"}, []string{"participant", "node_id", "host", "device_index"}),
		NodeGPUDeviceTemp:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_temperature_celsius", Help: "Per-device GPU temperature °C"}, []string{"participant", "node_id", "host", "device_index"}),
		NodeGPUDeviceMemTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_memory_total_mb", Help: "Per-device GPU total memory MB"}, []string{"participant", "node_id", "host", "device_index"}),
		NodeGPUDeviceMemFree:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_memory_free_mb", Help: "Per-device GPU free memory MB"}, []string{"participant", "node_id", "host", "device_index"}),
		NodeGPUDeviceMemUsed:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_memory_used_mb", Help: "Per-device GPU used memory MB"}, []string{"participant", "node_id", "host", "device_index"}),
		NodeGPUDeviceAvail:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_device_available", Help: "Per-device GPU available (1=yes 0=no)"}, []string{"participant", "node_id", "host", "device_index"}),

		// Node ML service state
		NodeServiceState:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_service_state", Help: "ML node service state (0=STOPPED 1=INFERENCE 2=POW 3=TRAIN)"}, []string{"participant", "node_id", "host"}),
		NodeDiskAvailableGB: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_disk_available_gb", Help: "ML node model cache available disk space GB"}, []string{"participant", "node_id", "host"}),

		// Network-wide
		NetParticipantWeight: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_network_participant_weight", Help: "Per-participant weight in active epoch"}, []string{"participant"}),
		NetNodePocWeight:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_network_node_poc_weight", Help: "Per-node PoC weight in active epoch"}, []string{"participant", "node_id"}),
		NetTotalWeight:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_network_total_weight", Help: "Total weight of all participants"}, []string{"participant"}),
		NetRewardPerWeight:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_network_reward_per_weight", Help: "Estimated GNK per unit of weight"}, []string{"participant"}),

		// Pricing / models
		PricingUoC:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_pricing_unit_of_compute_price", Help: "Unit of compute price"}),
		PricingDynamic:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_pricing_dynamic_enabled", Help: "Dynamic pricing enabled (1/0)"}),
		ModelPrice:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_pricing_model_price_per_token", Help: "Price per token per model"}, []string{"model_id"}),
		ModelUnits:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_pricing_model_units_per_token", Help: "Compute units per token"}, []string{"model_id"}),
		ModelVRAM:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_model_v_ram", Help: "VRAM (GB)"}, []string{"model_id"}),
		ModelThroughput: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_model_throughput_per_nonce", Help: "Throughput per nonce"}, []string{"model_id"}),
		ModelValThresh:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_model_validation_threshold", Help: "Validation threshold"}, []string{"model_id"}),

		// Participant live (current epoch)
		ParticipantEpochsDone:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_epochs_completed", Help: "Personal epochs completed"}, []string{"participant"}),
		ParticipantCoinBalance: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_coin_balance", Help: "Coin balance (internal points)"}, []string{"participant"}),
		ParticipantWallet:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_wallet_balance_gonka", Help: "Wallet balance in GNK (ngonka/1e9)"}, []string{"participant"}),
		ParticipantInferences:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_inference_count", Help: "Inferences in current epoch"}, []string{"participant"}),
		ParticipantMissed:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_missed_requests", Help: "Missed requests in current epoch"}, []string{"participant"}),
		ParticipantEarnedCoins: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_earned_coins", Help: "Earned coins in current epoch"}, []string{"participant"}),
		ParticipantValidated:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_validated_inferences", Help: "Validated inferences in current epoch"}, []string{"participant"}),
		ParticipantInvalidated: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_invalidated_inferences", Help: "Invalidated inferences in current epoch"}, []string{"participant"}),

		// Participant health (extended)
		ParticipantStatus:         prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_status", Help: "Participant status (0=UNSPECIFIED 1=ACTIVE 2=INACTIVE 3=INVALID 5=UNCONFIRMED)"}, []string{"participant"}),
		ParticipantConsecutiveInv: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_consecutive_invalid_inferences", Help: "Consecutive invalid inferences counter"}, []string{"participant"}),
		ParticipantBurnedCoins:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_burned_coins", Help: "Burned (penalized) coins in current epoch"}, []string{"participant"}),
		ParticipantRewardedCoins:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_rewarded_coins", Help: "Rewarded coins in current epoch (after distribution)"}, []string{"participant"}),
		ParticipantReputation:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_participant_reputation", Help: "Participant reputation score from epoch group data"}, []string{"participant"}),

		// Network — counts and epoch-level
		NetActiveParticipantCount: prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_network_active_participant_count", Help: "Number of participants with a non-empty address in the current epoch response"}),
		NetTotalParticipantCount:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_network_total_participant_count", Help: "Total number of participants in current epoch"}),
		NetEpochInferenceCount:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_network_epoch_inference_count", Help: "Total network inferences in current epoch"}, []string{"participant"}),

		// BLS DKG phase
		BLSDKGPhase:          prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_bls_dkg_phase", Help: "BLS DKG phase (0=UNDEFINED 1=DEALING 2=VERIFYING 3=COMPLETED 4=FAILED 5=SIGNED)"}, []string{"participant"}),
		BLSDealingDeadline:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_bls_dealing_deadline_block", Help: "BLS dealing phase deadline block height"}, []string{"participant"}),
		BLSVerifyingDeadline: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_bls_verifying_deadline_block", Help: "BLS verifying phase deadline block height"}, []string{"participant"}),

		// Model utilization / capacity
		ModelUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_model_utilization_percent", Help: "Model utilization % (inferences/capacity)"}, []string{"model_id"}),
		ModelCapacity:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_model_capacity", Help: "Model maximum capacity (AI tokens/epoch)"}, []string{"model_id"}),

		// Epoch history (label epoch = chain epoch number as string)
		EpochInferences:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_inference_count", Help: "Inferences for epoch N"}, []string{"participant", "epoch"}),
		EpochMissed:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_missed_requests", Help: "Missed requests for epoch N"}, []string{"participant", "epoch"}),
		EpochEarnedCoins: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_earned_coins", Help: "Earned coins (internal points) for epoch N"}, []string{"participant", "epoch"}),
		EpochValidated:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_validated_inferences", Help: "Validated inferences for epoch N"}, []string{"participant", "epoch"}),
		EpochInvalidated: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_invalidated_inferences", Help: "Invalidated inferences for epoch N"}, []string{"participant", "epoch"}),
		EpochCoinBalance: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_coin_balance", Help: "Coin balance at epoch end"}, []string{"participant", "epoch"}),
		EpochDone:        prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_epochs_completed", Help: "Personal epochs completed at epoch end"}, []string{"participant", "epoch"}),
		EpochMissRate:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_miss_rate_percent", Help: "Miss rate %% for epoch N"}, []string{"participant", "epoch"}),
		EpochPocWeight:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_poc_weight", Help: "PoC weight for epoch N"}, []string{"participant", "epoch", "node_id"}),
		EpochTimeslot:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_timeslot_assigned", Help: "Timeslot assigned in epoch N (1/0)"}, []string{"participant", "epoch"}),
		EpochStartTime:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_start_time", Help: "Epoch start unix timestamp"}, []string{"participant", "epoch"}),
		EpochEndTime:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_end_time", Help: "Epoch end unix timestamp (estimated for live)"}, []string{"participant", "epoch"}),
		EpochDuration:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_duration_seconds", Help: "Real epoch duration (seconds)"}, []string{"participant", "epoch"}),
		EpochEarnedGNK:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_earned_gonka", Help: "Wallet balance delta for epoch (GNK)"}, []string{"participant", "epoch"}),
		EpochRewardedGNK: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_rewarded_gonka", Help: "On-chain rewarded GNK for epoch N (rewarded_coins/1e9)"}, []string{"participant", "epoch"}),
		EpochClaimed:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_claimed", Help: "1 if epoch reward was claimed, 0 if not"}, []string{"participant", "epoch"}),
		EpochEstimated:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_epoch_estimated_reward_gonka", Help: "Estimated GNK reward for epoch N (weight × emission/total_weight)"}, []string{"participant", "epoch"}),

		// Stats — network-wide inference statistics
		StatsAiTokens:        prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_stats_ai_tokens_total", Help: "Total AI tokens processed network-wide (cumulative)"}),
		StatsInferences:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_stats_inferences_total", Help: "Total inferences processed network-wide (cumulative)"}),
		StatsActualCost:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_stats_actual_cost_total", Help: "Total actual inference cost network-wide (coins, cumulative)"}),
		StatsModelAiTokens:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_stats_model_ai_tokens", Help: "AI tokens processed per model (cumulative)"}, []string{"model_id"}),
		StatsModelInferences: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_stats_model_inferences", Help: "Inferences processed per model (cumulative)"}, []string{"model_id"}),

		// Bridge — queue status for the Gonka bridge
		BridgePendingBlocks:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_bridge_pending_blocks", Help: "Bridge pending block count"}),
		BridgePendingReceipts: prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_bridge_pending_receipts", Help: "Bridge pending receipt count"}),
		BridgeReadyToProcess:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_bridge_ready_to_process", Help: "Bridge ready to process (1=yes 0=no)"}),
		BridgeEarliestBlock:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_bridge_earliest_block_number", Help: "Bridge earliest pending block number"}),
		BridgeLatestBlock:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_bridge_latest_block_number", Help: "Bridge latest pending block number"}),

		// Node managers — running/healthy state per ML node manager
		NodeManagerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_manager_running", Help: "ML node manager running (1=yes 0=no)"}, []string{"participant", "node_id", "host", "manager"}),
		NodeManagerHealthy: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_manager_healthy", Help: "ML node manager healthy (1=yes 0=no)"}, []string{"participant", "node_id", "host", "manager"}),

		// GPU driver info — version in labels, value always 1
		NodeGPUDriverInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_node_gpu_driver_info", Help: "GPU driver info (value=1, version in labels)"}, []string{"participant", "node_id", "host", "driver_version", "cuda_version"}),

		// Tokenomics — chain-wide token flow counters
		TokenomicsTotalFees:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_tokenomics_total_fees", Help: "Total fees collected (ngonka)"}),
		TokenomicsTotalSubsidies: prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_tokenomics_total_subsidies", Help: "Total subsidies issued (ngonka)"}),
		TokenomicsTotalRefunded:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_tokenomics_total_refunded", Help: "Total amount refunded (ngonka)"}),
		TokenomicsTotalBurned:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "gonka_tokenomics_total_burned", Help: "Total amount burned (ngonka)"}),

		// PoC v2 — proof-of-compute artifact and weight data
		PoCv2ArtifactCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_poc_v2_artifact_count", Help: "PoC v2 artifact count from store commit"}, []string{"participant"}),
		PoCv2NodeWeight:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "gonka_poc_v2_node_weight", Help: "PoC v2 per-node weight distribution"}, []string{"participant", "node_id"}),
	}

	reg.MustRegister(
		m.BlockHeight,
		m.BlockHeightMax,
		m.BlockTimeLocal,
		m.BlockTimeNetwork,
		m.CatchingUp,
		m.ChainEpoch,
		m.NodeStatus,
		m.NodeIntended,
		m.PocCurrent,
		m.PocIntended,
		m.NodePocWeight,
		m.NodeTimeslot,
		m.NodeGPUCount,
		m.NodeGPUUtil,
		m.NodeHardwareInfo,
		m.NodeGPUDeviceUtil,
		m.NodeGPUDeviceTemp,
		m.NodeGPUDeviceMemTotal,
		m.NodeGPUDeviceMemFree,
		m.NodeGPUDeviceMemUsed,
		m.NodeGPUDeviceAvail,
		m.NodeServiceState,
		m.NodeDiskAvailableGB,
		m.NetParticipantWeight,
		m.NetNodePocWeight,
		m.NetTotalWeight,
		m.NetRewardPerWeight,
		m.PricingUoC,
		m.PricingDynamic,
		m.ModelPrice,
		m.ModelUnits,
		m.ModelVRAM,
		m.ModelThroughput,
		m.ModelValThresh,
		m.ParticipantEpochsDone,
		m.ParticipantCoinBalance,
		m.ParticipantWallet,
		m.ParticipantInferences,
		m.ParticipantMissed,
		m.ParticipantEarnedCoins,
		m.ParticipantValidated,
		m.ParticipantInvalidated,
		m.ParticipantStatus,
		m.ParticipantConsecutiveInv,
		m.ParticipantBurnedCoins,
		m.ParticipantRewardedCoins,
		m.ParticipantReputation,
		m.NetActiveParticipantCount,
		m.NetTotalParticipantCount,
		m.NetEpochInferenceCount,
		m.BLSDKGPhase,
		m.BLSDealingDeadline,
		m.BLSVerifyingDeadline,
		m.ModelUtilization,
		m.ModelCapacity,
		m.EpochInferences,
		m.EpochMissed,
		m.EpochEarnedCoins,
		m.EpochValidated,
		m.EpochInvalidated,
		m.EpochCoinBalance,
		m.EpochDone,
		m.EpochMissRate,
		m.EpochPocWeight,
		m.EpochTimeslot,
		m.EpochStartTime,
		m.EpochEndTime,
		m.EpochDuration,
		m.EpochEarnedGNK,
		m.EpochRewardedGNK,
		m.EpochClaimed,
		m.EpochEstimated,
		m.StatsAiTokens,
		m.StatsInferences,
		m.StatsActualCost,
		m.StatsModelAiTokens,
		m.StatsModelInferences,
		m.BridgePendingBlocks,
		m.BridgePendingReceipts,
		m.BridgeReadyToProcess,
		m.BridgeEarliestBlock,
		m.BridgeLatestBlock,
		m.NodeManagerRunning,
		m.NodeManagerHealthy,
		m.NodeGPUDriverInfo,
		m.TokenomicsTotalFees,
		m.TokenomicsTotalSubsidies,
		m.TokenomicsTotalRefunded,
		m.TokenomicsTotalBurned,
		m.PoCv2ArtifactCount,
		m.PoCv2NodeWeight,
	)

	return m
}
