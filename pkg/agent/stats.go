package agent

import "time"

// AgentStatsRecord holds a single stats sample for an agent.
type AgentStatsRecord struct {
	CollectedAt  time.Time `json:"collected_at"`
	AgentName    string    `json:"agent_name"`
	CPUPct       float64   `json:"cpu_pct"`
	MemUsedMB    float64   `json:"mem_used_mb"`
	MemLimitMB   float64   `json:"mem_limit_mb"`
	NetRxMB      float64   `json:"net_rx_mb"`
	NetTxMB      float64   `json:"net_tx_mb"`
	BlockReadMB  float64   `json:"block_read_mb"`
	BlockWriteMB float64   `json:"block_write_mb"`
}
