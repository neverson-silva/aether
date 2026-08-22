package domain

type Net struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type Disk struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Percent    float64 `json:"percent"`
}

type Stats struct {
	CPUPercent float64   `json:"cpu_percent"`
	CPUCores   int       `json:"cpu_cores"`
	MemTotal   uint64    `json:"mem_total"`
	MemUsed    uint64    `json:"mem_used"`
	MemPercent float64   `json:"mem_percent"`
	Net        Net       `json:"net"`
	Disk       Disk      `json:"disk"`
	Uptime     uint64    `json:"uptime"`
	Load       []float64 `json:"load"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	// Source identifies where the metrics were collected: "host-agent"
	// (real host) or "runtime" (the container runtime/VM the API runs in).
	Source string `json:"source"`
	// RuntimeCores is the CPU count of the container runtime (VM), used to
	// normalize container CPU percentages. It differs from CPUCores when the
	// host agent reports the physical machine.
	RuntimeCores int `json:"runtime_cores"`
}

type Event struct {
	TS     string `json:"ts"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}
