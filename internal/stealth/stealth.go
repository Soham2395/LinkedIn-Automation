package stealth

import (
	"linkedin-automation/internal/action"
	"math/rand"
	"time"
)

type Profile struct {
	SpeedMultiplier float64 
	JitterFactor    float64 
}

type Engine struct {
	rng     *rand.Rand
	profile Profile
}

func NewEngine(profile Profile) *Engine {
	return &Engine{
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		profile: profile,
	}
}

func (e *Engine) Before(act action.Action) {
	switch act.Type {
	case action.ActionSearchProfiles:
		e.shortHesitation()
	case action.ActionPaginateSearch:
		e.mediumPause()
	case action.ActionVisitProfile:
		e.shortHesitation()
	case action.ActionSendConnection:
		e.mediumPause()
	case action.ActionSendMessage:
		e.longThink()
	default:
		e.shortHesitation()
	}
}

func (e *Engine) After(act action.Action) {
	switch act.Type {
	case action.ActionSearchProfiles:
		e.mediumPause()
	case action.ActionPaginateSearch:
		e.mediumPause()
	case action.ActionVisitProfile:
		e.longThink()
	case action.ActionSendConnection:
		e.shortHesitation()
	case action.ActionSendMessage:
		e.mediumPause()
	default:
		e.shortHesitation()
	}
}

func (e *Engine) shortHesitation() {
	base := 500 * time.Millisecond
	e.sleepWithJitter(base)
}

func (e *Engine) mediumPause() {
	base := 2 * time.Second
	e.sleepWithJitter(base)
}

func (e *Engine) longThink() {
	base := 5 * time.Second
	e.sleepWithJitter(base)
}

func (e *Engine) sleepWithJitter(base time.Duration) {
	adjustedBase := time.Duration(float64(base) * e.profile.SpeedMultiplier)
	jitterRange := float64(adjustedBase) * e.profile.JitterFactor
	jitter := time.Duration((e.rng.Float64()*2 - 1) * jitterRange)

	finalDelay := adjustedBase + jitter
	if finalDelay < 0 {
		finalDelay = 0
	}

	time.Sleep(finalDelay)
}
