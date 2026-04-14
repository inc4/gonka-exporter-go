# gonka-exporter-go

Prometheus exporter for Gonka network nodes. Collects metrics from the Gonka blockchain, inference nodes, and network participants.

## Quick Start

```bash
cp .env.example .env
nano .env  # fill in your values

docker compose -f docker-compose.image.yml up -d
```

Metrics available at: `http://localhost:9404/metrics`  
Health check: `http://localhost:9404/healthz`

## Configuration

All configuration is done via environment variables (`.env` file).

| Variable | Default | Description |
|---|---|---|
| `PARTICIPANT_ADDRESS` | — | Your Gonka wallet address (**required**) |
| `EXPORTER_PORT` | `9404` | Port the exporter listens on |
| `REFRESH_INTERVAL` | `30` | Metrics collection interval (seconds) |
| `NODE_RPC_URL` | `http://node:26657` | Tendermint RPC endpoint |
| `NODE_REST_URL` | `http://node:1317` | Cosmos SDK REST endpoint |
| `API_URL` | `http://api:9000` | Public API endpoint |
| `ADMIN_API_URL` | `http://api:9200` | Admin API endpoint (node management) |
| `GONKA_NETWORK` | `gonka-node_default` | Docker network name of the Gonka node stack |
| `MAX_EPOCH_HISTORY` | `500` | Number of epochs to keep in history file |
| `EPOCH_HISTORY_FILE` | `/data/epoch_history.json` | Path to epoch history file |
| `STATE_FILE` | `/data/exporter_state.json` | Path to current epoch state file |

## Metrics

### Chain / Sync

| Metric | Labels | Description |
|---|---|---|
| `gonka_block_height` | participant | Latest block height from local node |
| `gonka_block_height_max` | participant | Maximum block height across public nodes |
| `gonka_block_time_seconds` | participant | Latest block timestamp (unix) |
| `gonka_chain_catching_up` | participant | 1 = syncing, 0 = synced |
| `gonka_chain_epoch` | participant | Current epoch number |

### Participant — Live (current epoch)

| Metric | Labels | Description |
|---|---|---|
| `gonka_participant_inference_count` | participant | Inferences in current epoch |
| `gonka_participant_missed_requests` | participant | Missed requests in current epoch |
| `gonka_participant_earned_coins` | participant | Earned coins (internal points) |
| `gonka_participant_rewarded_coins` | participant | Rewarded coins after distribution |
| `gonka_participant_burned_coins` | participant | Burned (penalized) coins |
| `gonka_participant_coin_balance` | participant | Internal coin balance |
| `gonka_participant_wallet_balance_gonka` | participant | Wallet balance in GNK |
| `gonka_participant_validated_inferences` | participant | Validated inferences |
| `gonka_participant_invalidated_inferences` | participant | Invalidated inferences |
| `gonka_participant_epochs_completed` | participant | Total epochs completed |
| `gonka_participant_status` | participant | Status: 0=UNSPECIFIED 1=ACTIVE 2=INACTIVE 3=INVALID 5=UNCONFIRMED |
| `gonka_participant_consecutive_invalid_inferences` | participant | Consecutive invalid inferences counter (health indicator) |
| `gonka_participant_reputation` | participant | Reputation score from epoch group data |

### Epoch History (label `epoch=N`)

| Metric | Labels | Description |
|---|---|---|
| `gonka_epoch_inference_count` | participant, epoch | Inferences for epoch N |
| `gonka_epoch_missed_requests` | participant, epoch | Missed requests for epoch N |
| `gonka_epoch_miss_rate_percent` | participant, epoch | Miss rate % for epoch N |
| `gonka_epoch_earned_coins` | participant, epoch | Earned coins for epoch N |
| `gonka_epoch_earned_gonka` | participant, epoch | Wallet delta for epoch N (GNK) |
| `gonka_epoch_rewarded_gonka` | participant, epoch | On-chain rewarded GNK for epoch N |
| `gonka_epoch_estimated_reward_gonka` | participant, epoch | Estimated GNK reward for epoch N |
| `gonka_epoch_claimed` | participant, epoch | 1 if epoch reward was claimed |
| `gonka_epoch_coin_balance` | participant, epoch | Coin balance at epoch end |
| `gonka_epoch_epochs_completed` | participant, epoch | Epochs completed at epoch end |
| `gonka_epoch_poc_weight` | participant, epoch, node_id | PoC weight for epoch N per node |
| `gonka_epoch_timeslot_assigned` | participant, epoch | Timeslot assigned in epoch N (1/0) |
| `gonka_epoch_start_time` | participant, epoch | Epoch start unix timestamp |
| `gonka_epoch_end_time` | participant, epoch | Epoch end unix timestamp (estimated for live) |
| `gonka_epoch_duration_seconds` | participant, epoch | Epoch duration in seconds |
| `gonka_epoch_validated_inferences` | participant, epoch | Validated inferences for epoch N |
| `gonka_epoch_invalidated_inferences` | participant, epoch | Invalidated inferences for epoch N |

### Network

| Metric | Labels | Description |
|---|---|---|
| `gonka_network_total_weight` | participant | Total weight of all participants |
| `gonka_network_reward_per_weight` | participant | Estimated GNK per unit of weight |
| `gonka_network_participant_weight` | participant | Per-participant weight (network-wide) |
| `gonka_network_node_poc_weight` | participant, node_id | Per-node PoC weight (network-wide) |
| `gonka_network_active_participant_count` | — | Number of active participants in current epoch |
| `gonka_network_total_participant_count` | — | Total number of participants in current epoch |
| `gonka_network_epoch_inference_count` | participant | Total network inferences in current epoch |

### BLS DKG

| Metric | Labels | Description |
|---|---|---|
| `gonka_bls_dkg_phase` | participant | DKG phase: 0=UNDEFINED 1=DEALING 2=VERIFYING 3=COMPLETED 4=FAILED 5=SIGNED |
| `gonka_bls_dealing_deadline_block` | participant | Block height deadline for dealing phase |
| `gonka_bls_verifying_deadline_block` | participant | Block height deadline for verifying phase |

### Node Hardware / Status

| Metric | Labels | Description |
|---|---|---|
| `gonka_node_status` | participant, node_id, host | Node status: 0=UNKNOWN 1=INFERENCE 2=POC 3=TRAINING 4=STOPPED 5=FAILED |
| `gonka_node_intended_status` | participant, node_id, host | Node intended status |
| `gonka_node_poc_current_status` | participant, node_id, host | PoC status: 0=IDLE 1=GENERATING 2=VALIDATING |
| `gonka_node_poc_intended_status` | participant, node_id, host | PoC intended status |
| `gonka_node_poc_weight` | participant, node_id, host, model | Current PoC weight per node per model |
| `gonka_node_poc_timeslot_assigned` | participant, node_id, host, model | Timeslot assigned (1/0) |
| `gonka_node_hardware_info` | participant, node_id, host, hardware_type, hardware_count | Hardware info (value=1, metadata in labels) |
| `gonka_node_service_state` | participant, node_id, host | ML node service state: 0=STOPPED 1=INFERENCE 2=POW 3=TRAIN |
| `gonka_node_disk_available_gb` | participant, node_id, host | ML node model cache available disk space (GB) |

### Node GPU — Aggregated

| Metric | Labels | Description |
|---|---|---|
| `gonka_node_gpu_device_count` | participant, node_id, host | Total GPU device count |
| `gonka_node_gpu_avg_utilization_percent` | participant, node_id, host | Average GPU utilization % across all devices |

### Node GPU — Per Device

| Metric | Labels | Description |
|---|---|---|
| `gonka_node_gpu_device_utilization_percent` | participant, node_id, host, device_index | Per-device GPU compute utilization % |
| `gonka_node_gpu_device_temperature_celsius` | participant, node_id, host, device_index | Per-device GPU temperature (°C) |
| `gonka_node_gpu_device_memory_total_mb` | participant, node_id, host, device_index | Per-device total GPU memory (MB) |
| `gonka_node_gpu_device_memory_used_mb` | participant, node_id, host, device_index | Per-device used GPU memory (MB) |
| `gonka_node_gpu_device_memory_free_mb` | participant, node_id, host, device_index | Per-device free GPU memory (MB) |
| `gonka_node_gpu_device_available` | participant, node_id, host, device_index | Per-device availability (1=OK, 0=unavailable) |

### Pricing / Models

| Metric | Labels | Description |
|---|---|---|
| `gonka_pricing_unit_of_compute_price` | — | Unit of compute price |
| `gonka_pricing_dynamic_enabled` | — | Dynamic pricing enabled (1/0) |
| `gonka_pricing_model_price_per_token` | model_id | Price per token per model |
| `gonka_pricing_model_units_per_token` | model_id | Compute units per token |
| `gonka_model_v_ram` | model_id | Model VRAM (GB) |
| `gonka_model_throughput_per_nonce` | model_id | Model throughput per nonce |
| `gonka_model_validation_threshold` | model_id | Model validation threshold |
| `gonka_model_utilization_percent` | model_id | Model utilization % (requires dynamic pricing enabled) |
| `gonka_model_capacity` | model_id | Model capacity in AI tokens/epoch (requires dynamic pricing enabled) |

## Running from source

```bash
git clone https://github.com/inc4/gonka-exporter-go
cd gonka-exporter-go

cp .env.example .env
nano .env

docker compose up -d --build
```

## Project Structure

```
gonka-exporter-go/
├── cmd/exporter/main.go        — entry point, HTTP server
├── internal/
│   ├── config/config.go        — configuration from env vars
│   ├── fetcher/fetcher.go      — HTTP requests to Gonka APIs (~500 lines, 20+ endpoints)
│   ├── metrics/metrics.go      — Prometheus metric definitions (100+ metrics)
│   ├── state/state.go          — persistent state and epoch history
│   └── collector/collector.go  — collection orchestration (~650 lines)
├── docker-compose.yml          — build from source
├── docker-compose.image.yml    — run from pre-built image
└── Dockerfile
```

## Epoch History

The exporter persists epoch history to `/data/epoch_history.json`. On restart, all historical metrics are restored automatically from this file. The current epoch state (start time, wallet balance at epoch start, block height) is persisted separately to `/data/exporter_state.json`.

To migrate history from a previous deployment:

```bash
docker run --rm \
  -v <old-volume>:/old:ro \
  -v gonka-exporter-go_epoch-data:/new \
  alpine sh -c "cp /old/epoch_history.json /new/epoch_history.json"

docker compose restart gonka-exporter
```

## Reward Estimation Formula

```
emission(N) = 323000 × exp(-0.000475 × (N - 1))
reward_per_weight = emission(N) / total_network_weight
estimated_reward  = your_weight × reward_per_weight
```

## Epoch Boundary Detection

On each collection cycle the exporter compares the current chain epoch against the previous value. When the epoch increments:

1. A full `EpochSnapshot` is saved to history (inferences, miss rate, earned/rewarded coins, PoC weights, wallet delta, estimated reward, on-chain claimed amount)
2. `EpochState` is updated with new epoch start values
3. All history metrics are re-registered with Prometheus from the snapshot

Epoch boundaries are annotated in Grafana via:
```promql
changes(gonka_participant_epochs_completed{participant="$participant"}[2m]) > 0
```
