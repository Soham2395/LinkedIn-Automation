package risk

import (
	"errors"
	"sync"
	"time"

	"linkedin-automation/internal/action"
)

var (
	ErrDailyRiskExceeded = errors.New("daily risk limit exceeded")
)

type ActionWeights map[action.ActionType]int

var DefaultWeights = ActionWeights{
	action.ActionLogin:            10,
	action.ActionRestoreSession:   2,
	action.ActionSearchProfiles:   5,
	action.ActionPaginateSearch:   2,
	action.ActionVisitProfile:     3,
	action.ActionSendConnection:   15,
	action.ActionDetectAcceptance: 1,
	action.ActionSendMessage:      8,
}

type Engine struct {
	mu             sync.RWMutex
	cumulativeRisk int
	maxDailyRisk   int
	currentDate    string
	weights        ActionWeights
}

func NewEngine(maxDailyRisk int) *Engine {
	return &Engine{
		maxDailyRisk: maxDailyRisk,
		currentDate:  today(),
		weights:      DefaultWeights,
	}
}

func NewEngineWithWeights(maxDailyRisk int, weights ActionWeights) *Engine {
	return &Engine{
		maxDailyRisk: maxDailyRisk,
		currentDate:  today(),
		weights:      weights,
	}
}

func (e *Engine) Apply(act action.Action) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.resetIfNewDay()

	weight := e.weightFor(act)

	if e.cumulativeRisk+weight > e.maxDailyRisk {
		return ErrDailyRiskExceeded
	}

	e.cumulativeRisk += weight
	return nil
}

func (e *Engine) Score() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cumulativeRisk
}

func (e *Engine) MaxDaily() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.maxDailyRisk
}

func (e *Engine) Remaining() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.maxDailyRisk - e.cumulativeRisk
}

func (e *Engine) WeightFor(actType action.ActionType) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if w, ok := e.weights[actType]; ok {
		return w
	}
	return 5
}

func (e *Engine) weightFor(act action.Action) int {
	if act.RiskWeight > 0 {
		return int(act.RiskWeight * 10)
	}
	if w, ok := e.weights[act.Type]; ok {
		return w
	}
	return 5
}

func (e *Engine) resetIfNewDay() {
	now := today()
	if now != e.currentDate {
		e.cumulativeRisk = 0
		e.currentDate = now
	}
}

func today() string {
	return time.Now().Format("2006-01-02")
}
