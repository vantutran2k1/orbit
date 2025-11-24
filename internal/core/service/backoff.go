package service

import (
	"math"
	"time"
)

const (
	DefaultMaxRetries = 3
	BaseDelay         = 10 * time.Second
)

func CalculateBackoff(retryCount int) time.Duration {
	if retryCount > 10 {
		retryCount = 10
	}

	factor := math.Pow(2, float64(retryCount))
	return BaseDelay * time.Duration(factor)
}
