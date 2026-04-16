package fetcher

// NodeEntry represents a single node returned by the admin API.
type NodeEntry struct {
	Node struct {
		ID      string `json:"id"`
		Host    string `json:"host"`
		PocPort int    `json:"poc_port"`
		Hardware []struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		} `json:"hardware"`
	} `json:"node"`
	State struct {
		CurrentStatus     string `json:"current_status"`
		IntendedStatus    string `json:"intended_status"`
		PocCurrentStatus  string `json:"poc_current_status"`
		PocIntendedStatus string `json:"poc_intended_status"`
		EpochMLNodes      map[string]struct {
			PocWeight          *int64 `json:"poc_weight"`
			TimeslotAllocation []bool `json:"timeslot_allocation"`
		} `json:"epoch_ml_nodes"`
	} `json:"state"`
}

func (h *HTTPFetcher) FetchNodes(adminURL string) ([]NodeEntry, error) {
	var nodes []NodeEntry
	err := get(adminURL+"/admin/v1/nodes", &nodes)
	return nodes, err
}
