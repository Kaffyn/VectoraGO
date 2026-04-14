package quant

import (
	"math"
	"math/rand"
)

// PolarQuant implementa extração de ângulos em coordenadas polares
// para compressão de vetores de alta dimensionalidade.
type PolarQuant struct {
	Dimension int
	RotationMatrix [][]float32
}

func NewPolarQuant(dim int) *PolarQuant {
	pq := &PolarQuant{
		Dimension: dim,
	}
	// TODO: Gerar matriz de rotação ortogonal estável (Householder ou QR)
	return pq
}

// Rotate aplica uma rotação aleatória estável para balancear a variância
func (pq *PolarQuant) Rotate(vector []float32) []float32 {
	if pq.RotationMatrix == nil {
		return vector // Pass-through se não inicializado
	}
	
	result := make([]float32, pq.Dimension)
	for i := 0; i < pq.Dimension; i++ {
		var sum float32
		for j := 0; j < pq.Dimension; j++ {
			sum += pq.RotationMatrix[i][j] * vector[j]
		}
		result[i] = sum
	}
	return result
}

// ExtractAngles extrai fases (ângulos) do vetor rotacionado
func (pq *PolarQuant) ExtractAngles(vector []float32) []float32 {
	angles := make([]float32, len(vector)/2)
	for i := 0; i < len(vector)-1; i += 2 {
		// atan2 extrai o ângulo no plano (xi, xi+1)
		angles[i/2] = float32(math.Atan2(float64(vector[i+1]), float64(vector[i])))
	}
	return angles
}
