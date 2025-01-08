package utils

// RSI calculates the Relative Strength Index for the given prices
func RSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	var rsi []float64
	gains := make([]float64, 0)
	losses := make([]float64, 0)

	// Calculate price changes
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change >= 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Calculate RSI
	for i := period; i <= len(gains); i++ {
		avgGain := average(gains[i-period : i])
		avgLoss := average(losses[i-period : i])

		if avgLoss == 0 {
			rsi = append(rsi, 100)
		} else {
			rs := avgGain / avgLoss
			rsi = append(rsi, 100-(100/(1+rs)))
		}
	}

	return rsi
}

func average(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}

	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return sum / float64(len(nums))
}
