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
