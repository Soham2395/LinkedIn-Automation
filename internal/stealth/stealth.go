package stealth

import (
	"linkedin-automation/internal/action"
	"linkedin-automation/internal/logger"
	"math/rand"
	"time"
)

type Engine struct {
	rng *rand.Rand
}

func NewEngine() *Engine {
	return &Engine{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) Before(act action.Action) {
	logger.Info("Stealth: Applying pre-action behavior", "action", act.Type)

	var baseDelay time.Duration

	switch act.Type {
	case action.ActionSearchProfiles:
		baseDelay = 1500 * time.Millisecond
	case action.ActionPaginateSearch:
		baseDelay = 1 * time.Second
	case action.ActionVisitProfile:
		baseDelay = 800 * time.Millisecond
	case action.ActionSendConnection:
		baseDelay = 2500 * time.Millisecond
	case action.ActionSendMessage:
		baseDelay = 4 * time.Second
	default:
		baseDelay = 500 * time.Millisecond
	}

	jitter := time.Duration(float64(baseDelay) * (0.8 + 0.4*e.rng.Float64()))
	time.Sleep(jitter)
}

func (e *Engine) After(act action.Action) {
	logger.Info("Stealth: Applying post-action behavior", "action", act.Type)

	var baseDelay time.Duration

	switch act.Type {
	case action.ActionSearchProfiles:
		baseDelay = 3 * time.Second
	case action.ActionPaginateSearch:
		baseDelay = 2 * time.Second
	case action.ActionVisitProfile:
		baseDelay = 8 * time.Second
	case action.ActionSendConnection:
		baseDelay = 1500 * time.Millisecond
	case action.ActionSendMessage:
		baseDelay = 3 * time.Second
	default:
		baseDelay = 1 * time.Second
	}

	jitter := time.Duration(float64(baseDelay) * (0.8 + 0.4*e.rng.Float64()))
	time.Sleep(jitter)
}
