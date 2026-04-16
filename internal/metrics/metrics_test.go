package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestNewMetrics_NoRegistrationPanic verifies that NewMetrics registers all
// metrics without panicking and returns a fully-initialised *Metrics.
// If any metric name is duplicated or invalid, MustRegister panics.
func TestNewMetrics_NoRegistrationPanic(t *testing.T) {
	reg := prometheus.NewRegistry()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewMetrics panicked: %v", r)
		}
	}()
	m := NewMetrics(reg)
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

// TestNewMetrics_IndependentRegistries verifies that two separate registries
// can each host a full Metrics set without collision — no shared global state.
func TestNewMetrics_IndependentRegistries(t *testing.T) {
	reg1 := prometheus.NewRegistry()
	reg2 := prometheus.NewRegistry()

	// Both must succeed without panic.
	m1 := NewMetrics(reg1)
	m2 := NewMetrics(reg2)

	if m1 == nil || m2 == nil {
		t.Fatal("one of the Metrics instances is nil")
	}

	// Write different values into each registry and confirm Gather succeeds.
	m1.ChainEpoch.WithLabelValues("addr1").Set(42)
	m2.ChainEpoch.WithLabelValues("addr1").Set(7)

	if _, err := reg1.Gather(); err != nil {
		t.Fatalf("reg1.Gather() error: %v", err)
	}
	if _, err := reg2.Gather(); err != nil {
		t.Fatalf("reg2.Gather() error: %v", err)
	}
}

// TestNewMetrics_NonVecGaugesExist verifies that plain (non-vec) Gauge metrics
// are immediately present in Gather output after registration — no values need
// to be set for them to appear.
func TestNewMetrics_NonVecGaugesExist(t *testing.T) {
	reg := prometheus.NewRegistry()
	NewMetrics(reg)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	names := make(map[string]bool, len(mfs))
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}

	// Spot-check a few well-known plain-Gauge metric names (non-vec).
	required := []string{
		"gonka_network_active_participant_count",
		"gonka_network_total_participant_count",
		"gonka_stats_ai_tokens_total",
		"gonka_bridge_pending_blocks",
		"gonka_tokenomics_total_fees",
	}
	for _, name := range required {
		if !names[name] {
			t.Errorf("metric %q not found in Gather output", name)
		}
	}
}

// TestNewMetrics_VecGaugeRecordsValue verifies that GaugeVec metrics work
// end-to-end: set a value, gather, confirm it's present with correct value.
func TestNewMetrics_VecGaugeRecordsValue(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.ChainEpoch.WithLabelValues("test_addr").Set(123)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	for _, mf := range mfs {
		if mf.GetName() == "gonka_chain_epoch" {
			for _, mm := range mf.GetMetric() {
				if mm.GetGauge().GetValue() == 123 {
					return // found
				}
			}
		}
	}
	t.Fatal("gonka_chain_epoch{participant=\"test_addr\"} = 123 not found in Gather output")
}
