package fetcher

import "fmt"

const mlNodeAPIBase = "/v3.0.8/api/v1"

// GPUDevice holds per-device GPU metrics.
type GPUDevice struct {
	Index              int
	UtilizationPercent float64
	TemperatureC       *float64
	TotalMemoryMB      *int64
	FreeMemoryMB       *int64
	UsedMemoryMB       *int64
	IsAvailable        bool
}

// GPUStats contains aggregated GPU info for a node.
type GPUStats struct {
	Count   int
	AvgUtil float64
	Devices []GPUDevice
}

func (h *HTTPFetcher) FetchGPUStats(host string, port int) GPUStats {
	var r struct {
		Devices []struct {
			Index              int      `json:"index"`
			UtilizationPercent float64  `json:"utilization_percent"`
			TemperatureC       *float64 `json:"temperature_c"`
			TotalMemoryMB      *int64   `json:"total_memory_mb"`
			FreeMemoryMB       *int64   `json:"free_memory_mb"`
			UsedMemoryMB       *int64   `json:"used_memory_mb"`
			IsAvailable        bool     `json:"is_available"`
		} `json:"devices"`
	}
	url := fmt.Sprintf("http://%s:%d%s/gpu/devices", host, port, mlNodeAPIBase)
	if err := get(url, &r); err != nil || len(r.Devices) == 0 {
		return GPUStats{}
	}
	var total float64
	devices := make([]GPUDevice, len(r.Devices))
	for i, d := range r.Devices {
		total += d.UtilizationPercent
		devices[i] = GPUDevice{
			Index:              d.Index,
			UtilizationPercent: d.UtilizationPercent,
			TemperatureC:       d.TemperatureC,
			TotalMemoryMB:      d.TotalMemoryMB,
			FreeMemoryMB:       d.FreeMemoryMB,
			UsedMemoryMB:       d.UsedMemoryMB,
			IsAvailable:        d.IsAvailable,
		}
	}
	return GPUStats{Count: len(r.Devices), AvgUtil: total / float64(len(r.Devices)), Devices: devices}
}

// FetchMLNodeState returns the current service state string ("POW", "INFERENCE", "TRAIN", "STOPPED").
func (h *HTTPFetcher) FetchMLNodeState(host string, port int) (string, error) {
	var r struct {
		State string `json:"state"`
	}
	url := fmt.Sprintf("http://%s:%d%s/state", host, port, mlNodeAPIBase)
	if err := get(url, &r); err != nil {
		return "", err
	}
	return r.State, nil
}

// FetchMLNodeDiskSpaceGB returns available disk space in GB for the model cache.
func (h *HTTPFetcher) FetchMLNodeDiskSpaceGB(host string, port int) (float64, error) {
	var r struct {
		AvailableGB float64 `json:"available_gb"`
	}
	url := fmt.Sprintf("http://%s:%d%s/models/space", host, port, mlNodeAPIBase)
	if err := get(url, &r); err != nil {
		return 0, err
	}
	return r.AvailableGB, nil
}

// MLNodeManagerStatus holds running/healthy status for one manager.
type MLNodeManagerStatus struct {
	Running bool
	Healthy bool
}

// MLNodeHealthData holds manager health for all ML node managers.
type MLNodeHealthData struct {
	ManagerPow       MLNodeManagerStatus
	ManagerInference MLNodeManagerStatus
	ManagerTrain     MLNodeManagerStatus
}

// FetchMLNodeHealth returns manager health. Tries /health first (root endpoint),
// then /v3.0.8/api/v1/health as fallback.
func (h *HTTPFetcher) FetchMLNodeHealth(host string, port int) (*MLNodeHealthData, error) {
	var r struct {
		Managers struct {
			Pow struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"pow"`
			Inference struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"inference"`
			Train struct {
				Running bool `json:"running"`
				Healthy bool `json:"healthy"`
			} `json:"train"`
		} `json:"managers"`
	}
	url := fmt.Sprintf("http://%s:%d/health", host, port)
	if err := get(url, &r); err != nil {
		// Fallback: versioned path
		url = fmt.Sprintf("http://%s:%d%s/health", host, port, mlNodeAPIBase)
		if err2 := get(url, &r); err2 != nil {
			return nil, err2
		}
	}
	return &MLNodeHealthData{
		ManagerPow:       MLNodeManagerStatus{Running: r.Managers.Pow.Running, Healthy: r.Managers.Pow.Healthy},
		ManagerInference: MLNodeManagerStatus{Running: r.Managers.Inference.Running, Healthy: r.Managers.Inference.Healthy},
		ManagerTrain:     MLNodeManagerStatus{Running: r.Managers.Train.Running, Healthy: r.Managers.Train.Healthy},
	}, nil
}

// GPUDriverData holds GPU driver and CUDA version strings.
type GPUDriverData struct {
	DriverVersion     string
	CudaDriverVersion string
}

// FetchGPUDriverInfo returns GPU driver info from /v3.0.8/api/v1/gpu/driver.
func (h *HTTPFetcher) FetchGPUDriverInfo(host string, port int) (*GPUDriverData, error) {
	var r struct {
		DriverVersion     string `json:"driver_version"`
		CudaDriverVersion string `json:"cuda_driver_version"`
	}
	url := fmt.Sprintf("http://%s:%d%s/gpu/driver", host, port, mlNodeAPIBase)
	if err := get(url, &r); err != nil {
		return nil, err
	}
	if r.DriverVersion == "" {
		return nil, fmt.Errorf("empty driver version")
	}
	return &GPUDriverData{DriverVersion: r.DriverVersion, CudaDriverVersion: r.CudaDriverVersion}, nil
}
