package pgxServerless

import (
	"math"
	"math/rand"
	"time"
)

type delayConfig struct {
	backoffCapMs   float32
	backoffBaseMs  float32
	backoffDelayMs float32
}

type delay struct {
	config delayConfig
}

func newDelay(config delayConfig) delay {
	rand.Seed(time.Now().UTC().UnixNano())
	return delay{
		config: config,
	}
}

func (d delay) getDecorrelatedJitterDelayMs() time.Duration {
	capMs := d.config.backoffCapMs
	base := d.config.backoffBaseMs
	backDelay := d.config.backoffDelayMs
	p1 := base - backDelay * 3 - 1
	p2 := rand.Float32() * p1
	randRange := math.Floor(float64(p2)) + float64(backDelay * 3)

	return time.Duration(math.Min(float64(capMs), randRange)) * time.Millisecond
}

func (d delay) getDelay() time.Duration {
	return d.getDecorrelatedJitterDelayMs()
}
