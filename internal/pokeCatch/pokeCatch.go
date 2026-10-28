package pokecatch

import ("math/rand/v2"

)

const (
	catchFloor   = 0.05
	catchCeiling = 0.95
)
func CatchChance (baseExperience int) float64{

	prob := 1.0/(1.0 + float64(baseExperience)/100)
	return min(max(prob, catchFloor), catchCeiling)
}

func Roll(baseExperience int) bool {
	return rand.Float64() < CatchChance(baseExperience)
}
