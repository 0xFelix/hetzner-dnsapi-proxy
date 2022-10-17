package hetzner

type Zone struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Zones struct {
	Zones []Zone `json:"zones"`
}

type Record struct {
	Id     *string `json:"id,omitempty"`
	Name   string  `json:"name"`
	TTL    int     `json:"ttl"`
	Type   string  `json:"type"`
	Value  string  `json:"value"`
	ZoneId string  `json:"zone_id"`
}

type Records struct {
	Records []Record `json:"records"`
}
