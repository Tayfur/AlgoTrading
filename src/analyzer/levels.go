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
	if len(candles) < 100 {
		return nil
	}

	// Calculate price zones with larger tolerance
	priceZones := make(map[float64]int)
	tolerance := 200.0 // Increased zone size for more stability

	// Use weighted touches for recent vs older candles
	recentWeight := 2.0 // Recent touches count more
	historyDepth := len(candles)
	midPoint := historyDepth / 2

	// Analyze price touches with time weighting
	for i, candle := range candles {
		weight := 1.0
		if i > midPoint {
			weight = recentWeight // More recent touches have higher weight
		}

		// Round prices to reduce noise
		highZone := math.Floor(candle.High/tolerance) * tolerance
		lowZone := math.Floor(candle.Low/tolerance) * tolerance

		priceZones[highZone] += int(weight)
		priceZones[lowZone] += int(weight)
	}

	// Convert zones to levels with stricter criteria
	var levels []PriceLevel
	currentPrice := candles[len(candles)-1].Close
	minTouches := 8 // Increased minimum touches for more stability

	// Filter significant levels
	for price, touches := range priceZones {
		if touches >= minTouches {
			strength := calculateLevelStrength(price, touches, candles)
			if strength >= 75 { // Higher strength threshold
				// Avoid levels too close to each other
				if !isNearExistingLevel(levels, price, tolerance*2) {
					level := PriceLevel{
						Price:      price,
						Type:       determineLevelType(price, currentPrice),
						Strength:   strength,
						TouchCount: touches,
					}
					levels = append(levels, level)
				}
			}
		}
	}

	// Sort by strength
	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Strength > levels[j].Strength
	})

	// Limit to top 2 levels of each type
	levels = limitLevelsByType(levels, 2)

	return levels
}

// Helper function to check if a new level is too close to existing ones
func isNearExistingLevel(levels []PriceLevel, price float64, minDistance float64) bool {
	for _, level := range levels {
		if math.Abs(level.Price-price) < minDistance {
			return true
		}
	}
	return false
}

// Helper function to limit the number of levels by type
func limitLevelsByType(levels []PriceLevel, maxPerType int) []PriceLevel {
	var supports, resistances []PriceLevel

	// Separate levels by type
	for _, level := range levels {
		if level.Type == "support" {
			supports = append(supports, level)
		} else {
			resistances = append(resistances, level)
		}
	}

	// Limit each type
	if len(supports) > maxPerType {
		supports = supports[:maxPerType]
	}
	if len(resistances) > maxPerType {
		resistances = resistances[:maxPerType]
	}

	// Combine and return
	return append(supports, resistances...)
}

func findPriceClusters(prices []float64) map[float64]int {
	clusters := make(map[float64]int)
	tolerance := 0.001 // 0.1% price range

	for _, price := range prices {
		roundedPrice := math.Floor(price/tolerance) * tolerance
		clusters[roundedPrice]++
	}

	return clusters
}

func calculateLevelStrength(price float64, touches int, candles []models.Candle) float64 {
	baseStrength := float64(touches) * 8

	// Count recent tests of the level
	recentCandles := candles[max(0, len(candles)-30):] // Last 30 candles
	recentTouches := 0

	for _, candle := range recentCandles {
		// Wider range for touch detection
		touchRange := price * 0.002 // 0.2% range
		if math.Abs(candle.High-price) <= touchRange ||
			math.Abs(candle.Low-price) <= touchRange {
			recentTouches++
		}
	}

	// Add strength for recent touches
	strength := baseStrength + float64(recentTouches)*15

	// Cap strength at 100
	return math.Min(strength, 100)
}

func determineLevelType(level, currentPrice float64) string {
	if level > currentPrice {
		return "resistance"
	}
	return "support"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
