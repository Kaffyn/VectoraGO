package quant

import (
	"math"
	"math/rand"
)

// OrthogonalRotator realiza uma rotação aleatória determinística nos vetores.
// Esta técnica é inspirada no ScaNN e outras implementações de Vector Quantization
// para balancear a variância entre as dimensões (spread energy).
type OrthogonalRotator struct {
	Dimension int
	Matrix    [][]float32
}

// NewOrthogonalRotator cria um rotacionador com uma matriz ortogonal gerada via QR.
// Usamos um seed para garantir que a rotação seja a mesma em toda a coleção.
func NewOrthogonalRotator(dim int, seed int64) *OrthogonalRotator {
	r := &OrthogonalRotator{
		Dimension: dim,
	}
	r.Matrix = generateRandomOrthogonalMatrix(dim, seed)
	return r
}

// Rotate aplica a transformação matricial: v' = R * v
func (r *OrthogonalRotator) Rotate(vector []float32) []float32 {
	if len(vector) != r.Dimension {
		return vector
	}

	result := make([]float32, r.Dimension)
	for i := 0; i < r.Dimension; i++ {
		var sum float32
		row := r.Matrix[i]
		for j := 0; j < r.Dimension; j++ {
			sum += row[j] * vector[j]
		}
		result[i] = sum
	}
	return result
}

// generateRandomOrthogonalMatrix cria uma matriz ortogonal usando o método de Gram-Schmidt.
// Para dimensões muito altas (>2048), métodos de Fast Walsh-Hadamard são preferíveis.
func generateRandomOrthogonalMatrix(dim int, seed int64) [][]float32 {
	rng := rand.New(rand.NewSource(seed))
	
	// 1. Iniciar com matriz de ruído Gaussiano
	matrix := make([][]float32, dim)
	for i := 0; i < dim; i++ {
		matrix[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			matrix[i][j] = float32(rng.NormFloat64())
		}
	}

	// 2. Processo de Ortogonalização de Gram-Schmidt
	for i := 0; i < dim; i++ {
		// Ortogonalizar em relação aos vetores anteriores
		for j := 0; j < i; j++ {
			dot := dotProduct(matrix[i], matrix[j])
			for k := 0; k < dim; k++ {
				matrix[i][k] -= dot * matrix[j][k]
			}
		}
		
		// Normalizar
		norm := float32(math.Sqrt(float64(dotProduct(matrix[i], matrix[i]))))
		if norm > 1e-8 {
			for k := 0; k < dim; k++ {
				matrix[i][k] /= norm
			}
		}
	}

	return matrix
}

func dotProduct(v1, v2 []float32) float32 {
	var sum float32
	for i := range v1 {
		sum += v1[i] * v2[i]
	}
	return sum
}
