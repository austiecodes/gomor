package decay

import (
	"math"
	"time"

	"github.com/austiecodes/gomor/internal/memory/memtypes"
)

const (
	explicitConfidence      = 0.90
	extractedConfidence     = 0.70
	explicitStabilityDays   = 30.0
	extractedStabilityDays  = 14.0
	baseFreshnessMultiplier = 0.70
	freshnessWeight         = 0.30
	reinforcementThreshold  = 0.55
	reinforcementFactor     = 1.05
	maxStabilityDays        = 180.0
)

func DefaultConfidence(source memtypes.MemorySource) float64 {
	if source == memtypes.SourceExtracted {
		return extractedConfidence
	}
	return explicitConfidence
}

func DefaultStabilityDays(source memtypes.MemorySource) float64 {
	if source == memtypes.SourceExtracted {
		return extractedStabilityDays
	}
	return explicitStabilityDays
}

func EffectiveLastRetrievedAt(item memtypes.MemoryItem) time.Time {
	if item.LastRetrievedAt != nil && !item.LastRetrievedAt.IsZero() {
		return item.LastRetrievedAt.UTC()
	}
	return item.CreatedAt.UTC()
}

func Freshness(now time.Time, lastRetrievedAt time.Time, stabilityDays float64) float64 {
	if stabilityDays <= 0 {
		stabilityDays = explicitStabilityDays
	}

	elapsedDays := now.UTC().Sub(lastRetrievedAt.UTC()).Hours() / 24
	if elapsedDays < 0 {
		elapsedDays = 0
	}

	return math.Pow(2, -(elapsedDays / stabilityDays))
}

func FinalScore(relevance float64, freshness float64, confidence float64) float64 {
	if confidence <= 0 {
		confidence = explicitConfidence
	}
	return relevance * (baseFreshnessMultiplier + freshnessWeight*freshness) * confidence
}

func ShouldReinforce(score float64) bool {
	return score >= reinforcementThreshold
}

func ReinforcedStability(stabilityDays float64) float64 {
	if stabilityDays <= 0 {
		stabilityDays = explicitStabilityDays
	}
	return math.Min(stabilityDays*reinforcementFactor, maxStabilityDays)
}
