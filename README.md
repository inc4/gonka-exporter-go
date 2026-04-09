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

## Metrics

### Chain / Sync

| Metric | Description |
|---|---|
| `gonka_block_height` | Latest block height from local node |
| `gonka_block_height_max` | Maximum block height across public nodes |
| `gonka_block_time_seconds` | Latest block timestamp (unix) |
| `gonka_chain_catching_up` | 1 = syncing, 0 = synced |
| `gonka_chain_epoch` | Current epoch number |

### Participant (live)

| Metric | Description |
|---|---|
| `gonka_participant_inference_count` | Inferences in current epoch |
| `gonka_participant_missed_requests` | Missed requests in current epoch |
| `gonka_participant_earned_coins` | Earned coins (internal points) |
| `gonka_participant_coin_balance` | Internal coin balance |
| `gonka_participant_wallet_balance_gonka` | Wallet balance in GNK |
| `gonka_participant_validated_inferences` | Validated inferences |
| `gonka_participant_invalidated_inferences` | Invalidated inferences |
| `gonka_participant_epochs_completed` | Total epochs completed |

### Epoch History (label `epoch=N`)

| Metric | Description |
|---|---|
| `gonka_epoch_inference_count` | Inferences for epoch N |
| `gonka_epoch_missed_requests` | Missed requests for epoch N |
| `gonka_epoch_miss_rate_percent` | Miss rate % for epoch N |
| `gonka_epoch_earned_coins` | Earned coins for epoch N |
| `gonka_epoch_earned_gonka` | Wallet delta for epoch N (GNK) |
| `gonka_epoch_rewarded_gonka` | On-chain rewarded GNK for epoch N |
| `gonka_epoch_estimated_reward_gonka` | Estimated GNK reward for epoch N |
| `gonka_epoch_claimed` | 1 if epoch reward was claimed |
| `gonka_epoch_coin_balance` | Coin balance at epoch end |
| `gonka_epoch_epochs_completed` | Epochs completed at epoch end |
| `gonka_epoch_poc_weight` | PoC weight for epoch N (per node) |
| `gonka_epoch_timeslot_assigned` | Timeslot assigned in epoch N |
| `gonka_epoch_start_time` | Epoch start unix timestamp |
| `gonka_epoch_end_time` | Epoch end unix timestamp |
| `gonka_epoch_duration_seconds` | Epoch duration in seconds |
| `gonka_epoch_validated_inferences` | Validated inferences for epoch N |
| `gonka_epoch_invalidated_inferences` | Invalidated inferences for epoch N |

### Network

| Metric | Description |
|---|---|
| `gonka_network_total_weight` | Total weight of all participants |
| `gonka_network_reward_per_weight` | Estimated GNK per unit of weight |
| `gonka_network_participant_weight` | Per-participant weight (network-wide) |
| `gonka_network_node_poc_weight` | Per-node PoC weight (network-wide) |

### Node Hardware / Status

| Metric | Description |
|---|---|
| `gonka_node_status` | Node status (0=UNKNOWN 1=INFERENCE 2=POC 3=TRAINING 4=STOPPED 5=FAILED) |
| `gonka_node_intended_status` | Node intended status |
| `gonka_node_poc_current_status` | PoC status (0=IDLE 1=GENERATING 2=VALIDATING) |
| `gonka_node_poc_weight` | Current PoC weight per node |
| `gonka_node_poc_timeslot_assigned` | Timeslot assigned (1/0) |
| `gonka_node_gpu_device_count` | GPU count |
| `gonka_node_gpu_avg_utilization_percent` | Average GPU utilization % |
| `gonka_node_hardware_info` | Hardware info (type, count) |

### Pricing / Models

| Metric | Description |
|---|---|
| `gonka_pricing_unit_of_compute_price` | Unit of compute price |
| `gonka_pricing_dynamic_enabled` | Dynamic pricing enabled (1/0) |
| `gonka_pricing_model_price_per_token` | Price per token per model |
| `gonka_model_v_ram` | Model VRAM (GB) |
| `gonka_model_throughput_per_nonce` | Model throughput per nonce |
| `gonka_model_validation_threshold` | Model validation threshold |

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
│   ├── fetcher/fetcher.go      — HTTP requests to Gonka APIs
│   ├── metrics/metrics.go      — Prometheus metric definitions
│   ├── state/state.go          — persistent state and epoch history
│   └── collector/collector.go  — collection logic
├── docker-compose.yml          — build from source
├── docker-compose.image.yml    — run from pre-built image
└── Dockerfile
```

## Epoch History

The exporter persists epoch history to `/data/epoch_history.json`. When restarted, all historical metrics are restored automatically from this file.

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
estimated_reward = your_weight × reward_per_weight
```
