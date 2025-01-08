package analyzer

import (
	"math"
	"sort"
	"trading-analyzer/src/models"
)

type PriceLevel struct {
	Price      float64
	Type       string  // "support" or "resistance"
	Strength   float64 // 0-100
	TouchCount int     // Number of times price touched this level
}

func FindSupportResistance(candles []models.Candle) []PriceLevel {
	var levels []PriceLevel

	// Need minimum candles for analysis
	if len(candles) < 100 {
		return levels
	}

	// Find price clusters
	pricePoints := make([]float64, 0)
	for _, c := range candles {
		pricePoints = append(pricePoints, c.High, c.Low)
	}

	clusters := findPriceClusters(pricePoints)

	// Analyze each cluster
	for price, touches := range clusters {
		if touches < 3 { // Minimum 3 touches to be significant
			continue
		}

		level := PriceLevel{
			Price:      price,
			TouchCount: touches,
		}

		// Determine if support or resistance
		level.Type = determineLevelType(price, candles[len(candles)-1].Close)

		// Calculate strength based on touches and recent tests
		level.Strength = calculateLevelStrength(price, touches, candles)

		levels = append(levels, level)
	}

	// Sort by strength
	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Strength > levels[j].Strength
	})

	// Return top 5 strongest levels
	if len(levels) > 5 {
		levels = levels[:5]
	}

	return levels
}

func findPriceClusters(prices []float64) map[float64]int {
	clusters := make(map[float64]int)

	// Group prices within 0.5% range
	tolerance := 0.005

	for _, price := range prices {
		found := false
		for existingPrice := range clusters {
			if math.Abs(price-existingPrice)/existingPrice < tolerance {
				clusters[existingPrice]++
				found = true
				break
			}
		}
		if !found {
			clusters[price] = 1
		}
	}

	return clusters
}

func determineLevelType(level, currentPrice float64) string {
	if level > currentPrice {
		return "resistance"
	}
	return "support"
}

func calculateLevelStrength(level float64, touches int, candles []models.Candle) float64 {
	strength := float64(touches) * 10 // Base strength from touches

	// Add strength based on recent tests
	recentCandles := candles[len(candles)-20:] // Last 20 candles
	recentTests := 0

	for _, c := range recentCandles {
		if math.Abs(c.High-level)/level < 0.005 || math.Abs(c.Low-level)/level < 0.005 {
			recentTests++
		}
	}

	strength += float64(recentTests) * 15

	// Cap at 100
	if strength > 100 {
		strength = 100
	}

	return strength
}
