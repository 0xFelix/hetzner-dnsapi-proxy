package hetzner

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Zones struct {
	Zones []Zone `json:"zones"`
}

type Record struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	TTL    int    `json:"ttl"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	ZoneID string `json:"zone_id"`
}

type Records struct {
	Records []Record `json:"records"`
}
