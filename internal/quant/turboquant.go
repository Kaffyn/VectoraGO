package quant

import (
	"fmt"
	"math"
)

// Quantizer define a interface para algoritmos de compressão de vetores
type Quantizer interface {
	Encode(vector []float32) ([]byte, error)
	Decode(data []byte) ([]float32, error)
	Config() QuantConfig
}

type QuantConfig struct {
	Type      string `json:"type"`       // "polar", "qjl", "default"
	Dimension int    `json:"dimension"`
	BitsPerDim int    `json:"bits_per_dim"`
}

// TurboQuant é o orquestrador de quantização avançada do Vectora
type TurboQuant struct {
	config  QuantConfig
	rotator *OrthogonalRotator
	qjl     *QJLQuant
}

func NewTurboQuant(cfg QuantConfig) *TurboQuant {
	// Usamos um seed fixo para o Beta; em produção isso pode ser per-tenant.
	seed := int64(42)
	return &TurboQuant{
		config:  cfg,
		rotator: NewOrthogonalRotator(cfg.Dimension, seed),
		qjl:     NewQJLQuant(cfg.Dimension, seed),
	}
}

func (t *TurboQuant) Encode(vector []float32) ([]byte, error) {
	if len(vector) != t.config.Dimension {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", t.config.Dimension, len(vector))
	}

	// 1. Rotação Ortogonal (Efeito ScaNN/Quark)
	// Espalha a informação uniformemente antes da quantização
	rotated := t.rotator.Rotate(vector)

	// 2. Estabilização QJL
	stabilized := t.qjl.Stabilize(rotated)

	// 3. Quantização 1-bit (Simplificado para Beta)
	return t.qjl.PackBits(stabilized), nil
}

func (t *TurboQuant) Decode(data []byte) ([]float32, error) {
	// No Beta, retornamos o vetor estabilizado/rotacionado (aproximação)
	// Para busca real Hamming, o Decode nem sempre é necessário se o índice busca bits
	vector := make([]float32, t.config.Dimension)
	
	for i := 0; i < t.config.Dimension; i++ {
		byteIdx := i / 8
		bitIdx := uint(i) % 8
		if (data[byteIdx] & (1 << bitIdx)) != 0 {
			vector[i] = 1.0
		} else {
			vector[i] = -1.0
		}
	}
	
	return vector, nil
}
