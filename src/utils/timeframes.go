package utils

import "time"

// Timeframe represents a trading timeframe.
type Timeframe struct {
	Duration time.Duration
	Label    string
}

// Predefined timeframes for analysis.
var Timeframes = []Timeframe{
	{Duration: 5 * time.Minute, Label: "5min"},
	{Duration: 30 * time.Minute, Label: "30min"},
	{Duration: 1 * time.Hour, Label: "1h"},
}

// ConvertToTimeframe converts a string label to a Timeframe.
func ConvertToTimeframe(label string) (Timeframe, bool) {
	for _, tf := range Timeframes {
		if tf.Label == label {
			return tf, true
		}
	}
	return Timeframe{}, false
}