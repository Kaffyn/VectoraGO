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
	config QuantConfig
}

func NewTurboQuant(cfg QuantConfig) *TurboQuant {
	return &TurboQuant{config: cfg}
}

func (t *TurboQuant) Encode(vector []float32) ([]byte, error) {
	// Implementação inicial: PolarQuant + Angle Extraction (Beta)
	// Para o MVP Beta, usaremos compressão de 1-bit para teste
	
	if len(vector) != t.config.Dimension {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", t.config.Dimension, len(vector))
	}

	// Simplificação: 1-bit sign quantization (Basic QJL style)
	// Futuras iterações incluirão Polar Rotation
	numBytes := (len(vector) + 7) / 8
	encoded := make([]byte, numBytes)
	
	for i, val := range vector {
		if val > 0 {
			encoded[i/8] |= (1 << (uint(i) % 8))
		}
	}
	
	return encoded, nil
}

func (t *TurboQuant) Decode(data []byte) ([]float32, error) {
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
