package stealth

import (
	"linkedin-automation/internal/action"
	"time"
)

func (e *Engine) ApplyImperfection(act action.Action) {
	if !e.profile.ImperfectionEnabled {
		return
	}

	if e.rng.Float64() > e.profile.TypoRate {
		return
	}

	switch act.Type {
	case action.ActionSendMessage:
		e.simulateTypo()
	case action.ActionSendConnection:
		e.simulateMouseMisalignment()
	case action.ActionSearchProfiles:
		e.simulateScrollOvershoot()
	default:
		time.Sleep(time.Duration(float64(500*time.Millisecond) * e.profile.DelayMultiplier))
	}
}

func (e *Engine) simulateTypo() {
	delay := e.profile.DelayMultiplier
	time.Sleep(time.Duration(float64(200*time.Millisecond) * delay))
	time.Sleep(time.Duration(float64(300*time.Millisecond) * delay))
	time.Sleep(time.Duration(float64(200*time.Millisecond) * delay))
}

func (e *Engine) simulateMouseMisalignment() {
	delay := e.profile.DelayMultiplier
	time.Sleep(time.Duration(float64(400*time.Millisecond) * delay))
	time.Sleep(time.Duration(float64(300*time.Millisecond) * delay))
	time.Sleep(time.Duration(float64(400*time.Millisecond) * delay))
}

func (e *Engine) simulateScrollOvershoot() {
	delay := e.profile.DelayMultiplier
	time.Sleep(time.Duration(float64(600*time.Millisecond) * delay))
	time.Sleep(time.Duration(float64(800*time.Millisecond) * delay))
}
