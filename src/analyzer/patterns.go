package analyzer

import (
	"trading-analyzer/src/models"
)

// AnalyzePatterns analyzes candle patterns to identify potential trading signals
func AnalyzePatterns(candles []models.Candle) []string {
	var patterns []string
	for i := 1; i < len(candles); i++ {
		if candles[i].Close > candles[i-1].Close {
			patterns = append(patterns, "bullish")
		} else if candles[i].Close < candles[i-1].Close {
			patterns = append(patterns, "bearish")
		}
	}
	return patterns
}

// IdentifyTradeSignal generates a trade signal based on analyzed patterns
func IdentifyTradeSignal(candles []Candle) string {
	// Placeholder for signal generation logic
	if len(candles) == 0 {
		return "No data available"
	}
	lastCandle := candles[len(candles)-1]
	if lastCandle.Close > lastCandle.Open {
		return "Buy"
	}
	return "Sell"
}

func patternMatches(pattern1, pattern2 []string) bool {
	if len(pattern1) != len(pattern2) {
		return false
	}
	for i := range pattern1 {
		if pattern1[i] != pattern2[i] {
			return false
		}
	}
	return true
}
