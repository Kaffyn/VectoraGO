package shared

import (
	"github.com/Kaffyn/Vectora/internal/core/engine"
	"github.com/Kaffyn/Vectora/internal/config/policies"
)

// CoreDeps injeta as dependencias nativas limpas para todos os Transportes (JSON-RPC, gRPC)
type CoreDeps struct {
	Engine *engine.Engine
	Policy *policies.PolicyEngine
}
