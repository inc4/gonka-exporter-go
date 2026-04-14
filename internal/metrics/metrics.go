package metrics

import "github.com/prometheus/client_golang/prometheus"

// Chain / sync
var (
	BlockHeight    = gaugeVec("gonka_block_height",      "Latest block height from local node",                       "participant")
	BlockHeightMax = gaugeVec("gonka_block_height_max",  "Maximum block height seen across public nodes",             "participant")
	BlockTime      = gaugeVec("gonka_block_time_seconds","Timestamp of latest block (unix)",                          "participant")
	CatchingUp     = gaugeVec("gonka_chain_catching_up", "1 = syncing, 0 = synced",                                  "participant")
	ChainEpoch     = gaugeVec("gonka_chain_epoch",       "Current global chain epoch number",                        "participant")
)

// Node hardware / status
var (
	NodeStatus       = gaugeVec("gonka_node_status",              "Node hardware status (0=UNKNOWN 1=INFERENCE 2=POC 3=TRAINING 4=STOPPED 5=FAILED)", "participant", "node_id", "host")
	NodeIntended     = gaugeVec("gonka_node_intended_status",     "Node intended status",                                                               "participant", "node_id", "host")
	PocCurrent       = gaugeVec("gonka_node_poc_current_status",  "PoC current status (0=IDLE 1=GENERATING 2=VALIDATING)",                             "participant", "node_id", "host")
	PocIntended      = gaugeVec("gonka_node_poc_intended_status", "PoC intended status",                                                               "participant", "node_id", "host")
	NodePocWeight    = gaugeVec("gonka_node_poc_weight",          "PoC weight per node per model",                                                     "participant", "node_id", "host", "model")
	NodeTimeslot     = gaugeVec("gonka_node_poc_timeslot_assigned","Timeslot assigned (1/0)",                                                          "participant", "node_id", "host", "model")
	NodeGPUCount     = gaugeVec("gonka_node_gpu_device_count",    "GPU device count",                                                                  "participant", "node_id", "host")
	NodeGPUUtil      = gaugeVec("gonka_node_gpu_avg_utilization_percent", "Average GPU utilization %",                                                 "participant", "node_id", "host")
	NodeHardwareInfo = gaugeVec("gonka_node_hardware_info",       "Hardware info (value=1, metadata in labels)",                                       "participant", "node_id", "host", "hardware_type", "hardware_count")
)

// Node GPU — per device
var (
	NodeGPUDeviceUtil    = gaugeVec("gonka_node_gpu_device_utilization_percent", "Per-device GPU compute utilization %",    "participant", "node_id", "host", "device_index")
	NodeGPUDeviceTemp    = gaugeVec("gonka_node_gpu_device_temperature_celsius",  "Per-device GPU temperature °C",           "participant", "node_id", "host", "device_index")
	NodeGPUDeviceMemTotal= gaugeVec("gonka_node_gpu_device_memory_total_mb",      "Per-device GPU total memory MB",          "participant", "node_id", "host", "device_index")
	NodeGPUDeviceMemFree = gaugeVec("gonka_node_gpu_device_memory_free_mb",       "Per-device GPU free memory MB",           "participant", "node_id", "host", "device_index")
	NodeGPUDeviceMemUsed = gaugeVec("gonka_node_gpu_device_memory_used_mb",       "Per-device GPU used memory MB",           "participant", "node_id", "host", "device_index")
	NodeGPUDeviceAvail   = gaugeVec("gonka_node_gpu_device_available",             "Per-device GPU available (1=yes 0=no)",   "participant", "node_id", "host", "device_index")
)

// Node ML service state
var (
	NodeServiceState     = gaugeVec("gonka_node_service_state",          "ML node service state (0=STOPPED 1=INFERENCE 2=POW 3=TRAIN)", "participant", "node_id", "host")
	NodeDiskAvailableGB  = gaugeVec("gonka_node_disk_available_gb",      "ML node model cache available disk space GB",                  "participant", "node_id", "host")
)

// Network-wide
var (
	NetParticipantWeight = gaugeVec("gonka_network_participant_weight", "Per-participant weight in active epoch",    "participant")
	NetNodePocWeight     = gaugeVec("gonka_network_node_poc_weight",    "Per-node PoC weight in active epoch",      "participant", "node_id")
	NetTotalWeight       = gaugeVec("gonka_network_total_weight",       "Total weight of all participants",         "participant")
	NetRewardPerWeight   = gaugeVec("gonka_network_reward_per_weight",  "Estimated GNK per unit of weight",         "participant")
)

// Pricing / models
var (
	PricingUoC      = gauge("gonka_pricing_unit_of_compute_price", "Unit of compute price")
	PricingDynamic  = gauge("gonka_pricing_dynamic_enabled",       "Dynamic pricing enabled (1/0)")
	ModelPrice      = gaugeVec("gonka_pricing_model_price_per_token", "Price per token per model",   "model_id")
	ModelUnits      = gaugeVec("gonka_pricing_model_units_per_token", "Compute units per token",     "model_id")
	ModelVRAM       = gaugeVec("gonka_model_v_ram",                   "VRAM (GB)",                   "model_id")
	ModelThroughput = gaugeVec("gonka_model_throughput_per_nonce",    "Throughput per nonce",         "model_id")
	ModelValThresh  = gaugeVec("gonka_model_validation_threshold",    "Validation threshold",         "model_id")
)

// Participant live (current epoch)
var (
	ParticipantEpochsDone  = gaugeVec("gonka_participant_epochs_completed",       "Personal epochs completed",              "participant")
	ParticipantCoinBalance = gaugeVec("gonka_participant_coin_balance",           "Coin balance (internal points)",         "participant")
	ParticipantWallet      = gaugeVec("gonka_participant_wallet_balance_gonka",   "Wallet balance in GNK (ngonka/1e9)",     "participant")
	ParticipantInferences  = gaugeVec("gonka_participant_inference_count",        "Inferences in current epoch",            "participant")
	ParticipantMissed      = gaugeVec("gonka_participant_missed_requests",        "Missed requests in current epoch",       "participant")
	ParticipantEarnedCoins = gaugeVec("gonka_participant_earned_coins",           "Earned coins in current epoch",          "participant")
	ParticipantValidated   = gaugeVec("gonka_participant_validated_inferences",   "Validated inferences in current epoch",  "participant")
	ParticipantInvalidated = gaugeVec("gonka_participant_invalidated_inferences", "Invalidated inferences in current epoch","participant")
)

// Participant health (extended)
var (
	ParticipantStatus           = gaugeVec("gonka_participant_status",                        "Participant status (0=UNSPECIFIED 1=ACTIVE 2=INACTIVE 3=INVALID 5=UNCONFIRMED)", "participant")
	ParticipantConsecutiveInv   = gaugeVec("gonka_participant_consecutive_invalid_inferences", "Consecutive invalid inferences counter",                                         "participant")
	ParticipantBurnedCoins      = gaugeVec("gonka_participant_burned_coins",                   "Burned (penalized) coins in current epoch",                                      "participant")
	ParticipantRewardedCoins    = gaugeVec("gonka_participant_rewarded_coins",                 "Rewarded coins in current epoch (after distribution)",                           "participant")
	ParticipantReputation       = gaugeVec("gonka_participant_reputation",                     "Participant reputation score from epoch group data",                             "participant")
)

// Network — counts and epoch-level
var (
	NetActiveParticipantCount = gauge("gonka_network_active_participant_count", "Number of active participants in current epoch")
	NetTotalParticipantCount  = gauge("gonka_network_total_participant_count",  "Total number of participants in current epoch")
	NetEpochInferenceCount    = gaugeVec("gonka_network_epoch_inference_count", "Total network inferences in current epoch", "participant")
)

// BLS DKG phase
var (
	BLSDKGPhase              = gaugeVec("gonka_bls_dkg_phase",               "BLS DKG phase (0=UNDEFINED 1=DEALING 2=VERIFYING 3=COMPLETED 4=FAILED 5=SIGNED)", "participant")
	BLSDealingDeadline       = gaugeVec("gonka_bls_dealing_deadline_block",   "BLS dealing phase deadline block height",                                          "participant")
	BLSVerifyingDeadline     = gaugeVec("gonka_bls_verifying_deadline_block", "BLS verifying phase deadline block height",                                        "participant")
)

// Model utilization / capacity
var (
	ModelUtilization = gaugeVec("gonka_model_utilization_percent", "Model utilization % (inferences/capacity)", "model_id")
	ModelCapacity    = gaugeVec("gonka_model_capacity",             "Model maximum capacity (AI tokens/epoch)",  "model_id")
)

// Epoch history (label epoch = chain epoch number as string)
var (
	EpochInferences  = gaugeVec("gonka_epoch_inference_count",        "Inferences for epoch N",                           "participant", "epoch")
	EpochMissed      = gaugeVec("gonka_epoch_missed_requests",        "Missed requests for epoch N",                      "participant", "epoch")
	EpochEarnedCoins = gaugeVec("gonka_epoch_earned_coins",           "Earned coins (internal points) for epoch N",       "participant", "epoch")
	EpochValidated   = gaugeVec("gonka_epoch_validated_inferences",   "Validated inferences for epoch N",                 "participant", "epoch")
	EpochInvalidated = gaugeVec("gonka_epoch_invalidated_inferences", "Invalidated inferences for epoch N",               "participant", "epoch")
	EpochCoinBalance = gaugeVec("gonka_epoch_coin_balance",           "Coin balance at epoch end",                        "participant", "epoch")
	EpochDone        = gaugeVec("gonka_epoch_epochs_completed",       "Personal epochs completed at epoch end",           "participant", "epoch")
	EpochMissRate    = gaugeVec("gonka_epoch_miss_rate_percent",      "Miss rate %% for epoch N",                         "participant", "epoch")
	EpochPocWeight   = gaugeVec("gonka_epoch_poc_weight",             "PoC weight for epoch N",                           "participant", "epoch", "node_id")
	EpochTimeslot    = gaugeVec("gonka_epoch_timeslot_assigned",      "Timeslot assigned in epoch N (1/0)",               "participant", "epoch")
	EpochStartTime   = gaugeVec("gonka_epoch_start_time",             "Epoch start unix timestamp",                       "participant", "epoch")
	EpochEndTime     = gaugeVec("gonka_epoch_end_time",               "Epoch end unix timestamp (estimated for live)",    "participant", "epoch")
	EpochDuration    = gaugeVec("gonka_epoch_duration_seconds",       "Real epoch duration (seconds)",                    "participant", "epoch")
	EpochEarnedGNK   = gaugeVec("gonka_epoch_earned_gonka",           "Wallet balance delta for epoch (GNK)",             "participant", "epoch")
	EpochRewardedGNK = gaugeVec("gonka_epoch_rewarded_gonka",         "On-chain rewarded GNK for epoch N (rewarded_coins/1e9)", "participant", "epoch")
	EpochClaimed     = gaugeVec("gonka_epoch_claimed",                "1 if epoch reward was claimed, 0 if not",          "participant", "epoch")
	EpochEstimated   = gaugeVec("gonka_epoch_estimated_reward_gonka", "Estimated GNK reward for epoch N (weight × emission/total_weight)", "participant", "epoch")
)

// Stats — network-wide inference statistics
var (
	StatsAiTokens        = gauge("gonka_stats_ai_tokens_total",     "Total AI tokens processed network-wide (cumulative)")
	StatsInferences      = gauge("gonka_stats_inferences_total",    "Total inferences processed network-wide (cumulative)")
	StatsActualCost      = gauge("gonka_stats_actual_cost_total",   "Total actual inference cost network-wide (coins, cumulative)")
	StatsModelAiTokens   = gaugeVec("gonka_stats_model_ai_tokens",  "AI tokens processed per model (cumulative)", "model_id")
	StatsModelInferences = gaugeVec("gonka_stats_model_inferences", "Inferences processed per model (cumulative)", "model_id")
)

// Bridge — queue status for the Gonka bridge
var (
	BridgePendingBlocks   = gauge("gonka_bridge_pending_blocks",        "Bridge pending block count")
	BridgePendingReceipts = gauge("gonka_bridge_pending_receipts",      "Bridge pending receipt count")
	BridgeReadyToProcess  = gauge("gonka_bridge_ready_to_process",      "Bridge ready to process (1=yes 0=no)")
	BridgeEarliestBlock   = gauge("gonka_bridge_earliest_block_number", "Bridge earliest pending block number")
	BridgeLatestBlock     = gauge("gonka_bridge_latest_block_number",   "Bridge latest pending block number")
)

// Node managers — running/healthy state per ML node manager
var (
	NodeManagerRunning = gaugeVec("gonka_node_manager_running", "ML node manager running (1=yes 0=no)", "participant", "node_id", "host", "manager")
	NodeManagerHealthy = gaugeVec("gonka_node_manager_healthy", "ML node manager healthy (1=yes 0=no)", "participant", "node_id", "host", "manager")
)

// GPU driver info — version in labels, value always 1
var (
	NodeGPUDriverInfo = gaugeVec("gonka_node_gpu_driver_info", "GPU driver info (value=1, version in labels)", "participant", "node_id", "host", "driver_version", "cuda_version")
)

// Tokenomics — chain-wide token flow counters
var (
	TokenomicsTotalFees      = gauge("gonka_tokenomics_total_fees",       "Total fees collected (ngonka)")
	TokenomicsTotalSubsidies = gauge("gonka_tokenomics_total_subsidies",  "Total subsidies issued (ngonka)")
	TokenomicsTotalRefunded  = gauge("gonka_tokenomics_total_refunded",   "Total amount refunded (ngonka)")
	TokenomicsTotalBurned    = gauge("gonka_tokenomics_total_burned",     "Total amount burned (ngonka)")
	TokenomicsTopRewardStart = gauge("gonka_tokenomics_top_reward_start", "Top reward start epoch index")
)


// PoC v2 — proof-of-compute artifact and weight data
var (
	PoCv2ArtifactCount = gaugeVec("gonka_poc_v2_artifact_count", "PoC v2 artifact count from store commit", "participant")
	PoCv2NodeWeight    = gaugeVec("gonka_poc_v2_node_weight",     "PoC v2 per-node weight distribution",    "participant", "node_id")
)

func gauge(name, help string) prometheus.Gauge {
	g := prometheus.NewGauge(prometheus.GaugeOpts{Name: name, Help: help})
	prometheus.MustRegister(g)
	return g
}

func gaugeVec(name, help string, labels ...string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labels)
	prometheus.MustRegister(g)
	return g
}
