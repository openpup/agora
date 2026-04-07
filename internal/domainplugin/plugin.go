package domainplugin

import (
	"context"

	"github.com/openpup/agora/internal/core"
)

type Plugin interface {
	Name() string
	Definition() core.DomainDef
	ValidateClaim(structured map[string]any) error
	Verify(ctx context.Context, signal core.Signal) (bool, map[string]any, error)
	ResolveConsensus(signals []core.Signal) (map[string]any, error)
}
