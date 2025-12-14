package risk

import (
	"errors"
	"sync"
)

var ErrRiskThresholdExceeded = errors.New("risk threshold exceeded")
type Engine struct {
	mu             sync.RWMutex
	cumulativeRisk int
	maxRisk        int
}

func NewEngine(maxRisk int) *Engine {
	return &Engine{
		maxRisk: maxRisk,
	}
}

func (e *Engine) Apply(weight int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cumulativeRisk+weight > e.maxRisk {
		return ErrRiskThresholdExceeded
	}

	e.cumulativeRisk += weight
	return nil
}

func (e *Engine) Score() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cumulativeRisk
}
