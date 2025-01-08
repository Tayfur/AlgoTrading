package analyzer

type Signal struct {
	Type      string
	Risk      float64
	Price     float64
	Strength  int
	TimeFrame string
}

func GenerateSignals(patterns []string, risk float64, currentPrice float64) []Signal {
	var signals []Signal
	patternCount := make(map[string]int)

	for _, pattern := range patterns {
		patternCount[pattern]++
	}

	// Generate signals only for strong patterns (appearing multiple times)
	for pattern, count := range patternCount {
		if count >= 3 { // Minimum pattern strength threshold
			signal := Signal{
				Price:    currentPrice,
				Risk:     risk,
				Strength: count,
			}

			switch pattern {
			case "bullish_engulfing":
				signal.Type = "buy"
			case "bearish_engulfing":
				signal.Type = "sell"
			default:
				continue
			}

			signals = append(signals, signal)
		}
	}

	// Limit to top 3 strongest signals
	if len(signals) > 3 {
		signals = signals[:3]
	}

	return signals
}
