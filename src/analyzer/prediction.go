package analyzer

import (
	"math"
	"trading-analyzer/src/models"
	"trading-analyzer/src/utils"
)

type PricePrediction struct {
	TimeFrame      string
	PredictedPrice float64
	Confidence     float64
	Direction      string // "up", "down", or "sideways"
}

func PredictPrice(candles []models.Candle, timeframe string) PricePrediction {
	if len(candles) < 50 {
		return PricePrediction{TimeFrame: timeframe}
	}

	// Get closing prices
	prices := make([]float64, len(candles))
	for i, c := range candles {
		prices[i] = c.Close
	}

	// Calculate technical indicators
	rsi := utils.RSI(prices, 14)
	lastRSI := rsi[len(rsi)-1]

	// Calculate moving averages
	ema20 := calculateEMA(prices, 20)
	ema50 := calculateEMA(prices, 50)

	// Calculate price momentum
	momentum := calculateMomentum(prices, 10)

	// Predict direction and price
	direction := predictDirection(lastRSI, ema20, ema50, momentum)
	predictedMove := calculatePredictedMove(candles, momentum)
	currentPrice := prices[len(prices)-1]

	prediction := PricePrediction{
		TimeFrame:      timeframe,
		PredictedPrice: currentPrice * (1 + predictedMove),
		Confidence:     calculateConfidence(lastRSI, ema20, ema50, momentum),
		Direction:      direction,
	}

	return prediction
}

func calculateEMA(prices []float64, period int) float64 {
	multiplier := 2.0 / float64(period+1)
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = (prices[i] * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

func calculateMomentum(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	return (prices[len(prices)-1] - prices[len(prices)-period]) / prices[len(prices)-period]
}

func predictDirection(rsi, ema20, ema50, momentum float64) string {
	if rsi > 70 && ema20 < ema50 {
		return "down"
	} else if rsi < 30 && ema20 > ema50 {
		return "up"
	} else if momentum > 0.02 {
		return "up"
	} else if momentum < -0.02 {
		return "down"
	}
	return "sideways"
}

func calculatePredictedMove(candles []models.Candle, momentum float64) float64 {
	volatility := calculateVolatility(candles)
	return momentum * volatility
}

func calculateVolatility(candles []models.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}

	returns := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		returns[i-1] = math.Log(candles[i].Close / candles[i-1].Close)
	}

	return standardDeviation(returns)
}

func standardDeviation(numbers []float64) float64 {
	mean := 0.0
	for _, n := range numbers {
		mean += n
	}
	mean /= float64(len(numbers))

	variance := 0.0
	for _, n := range numbers {
		variance += math.Pow(n-mean, 2)
	}
	variance /= float64(len(numbers))

	return math.Sqrt(variance)
}

func calculateConfidence(rsi, ema20, ema50, momentum float64) float64 {
	confidence := 50.0 // Base confidence

	// Adjust based on RSI extremes
	if rsi > 70 || rsi < 30 {
		confidence += 10
	}

	// Adjust based on EMA crossover
	if math.Abs(ema20-ema50) < ema20*0.001 {
		confidence += 15
	}

	// Adjust based on momentum
	if math.Abs(momentum) > 0.02 {
		confidence += 15
	}

	return math.Min(confidence, 100)
}
