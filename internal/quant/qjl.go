package quant

import (
	"math/rand"
)

// QJLQuant implementa Quantized Johnson-Lindenstrauss
// para estabilização de projeções de 1-bit.
type QJLQuant struct {
	Dimension int
	Seed      int64
}

func NewQJLQuant(dim int, seed int64) *QJLQuant {
	return &QJLQuant{
		Dimension: dim,
		Seed:      seed,
	}
}

// Stabilize aplica um bias fixo baseado em hash para evitar instabilidade
// em projeções de baixa fidelidade (1-bit).
func (q *QJLQuant) Stabilize(vector []float32) []float32 {
	// Deterministia baseada em seed e posição
	rng := rand.New(rand.NewSource(q.Seed))
	
	result := make([]float32, len(vector))
	for i := range vector {
		// Adiciona um ruído ortogonal controlado para "espalhar" os vetores
		// e melhorar a separabilidade após a quantização de 1-bit.
		bias := (rng.Float32() * 2) - 1 // [-1, 1]
		result[i] = vector[i] + (bias * 0.01) // 1% bias stabilization
	}
	return result
}

// PackBits converte o vetor quantizado de 1-bit em bytes compactos
func (q *QJLQuant) PackBits(vector []float32) []byte {
	numBytes := (len(vector) + 7) / 8
	packed := make([]byte, numBytes)
	for i, v := range vector {
		if v > 0 {
			packed[i/8] |= (1 << (uint(i) % 8))
		}
	}
	return packed
}
