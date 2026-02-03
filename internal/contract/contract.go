package contract

import "time"

type Enforcement string

const (
	EnforcementLenient  Enforcement = "lenient"
	EnforcementModerate Enforcement = "moderate"
	EnforcementStrict   Enforcement = "strict"
)

type Contract struct {
	RouteID       string            `json:"route_id"`
	Policy        string            `json:"policy"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Samples       int               `json:"samples"`
	Methods       map[string]bool   `json:"methods"`
	ContentTypes  map[string]bool   `json:"content_types"`
	QueryParams   map[string]bool   `json:"query_params"`
	HeaderNames   map[string]bool   `json:"header_names"`
	MaxBodyBytes  int64             `json:"max_body_bytes"`
	ObservedMax   int64             `json:"observed_max_body_bytes"`
}

func New(routeID, policy string) *Contract {
	return &Contract{
		RouteID:      routeID,
		Policy:       policy,
		GeneratedAt:  time.Now().UTC(),
		Methods:      map[string]bool{},
		ContentTypes: map[string]bool{},
		QueryParams:  map[string]bool{},
		HeaderNames:  map[string]bool{},
	}
}

func (c *Contract) Finalize(marginBytes int64) {
	if marginBytes < 0 {
		marginBytes = 0
	}
	c.MaxBodyBytes = c.ObservedMax + marginBytes
}
