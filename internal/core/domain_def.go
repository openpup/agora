package core

import "time"

type DomainDef struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Namespace   string         `json:"namespace"`
	ClaimSchema map[string]any `json:"claim_schema"`
	Resolution  Resolution     `json:"resolution"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
}
