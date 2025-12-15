package stealth

import (
	"linkedin-automation/internal/action"
	"linkedin-automation/internal/logger"
	"time"
)

// ApplyImperfection simulates human error and correction.
// It introduces visible delays and logging to represent mistakes like typos or mouse slips.
func (e *Engine) ApplyImperfection(act action.Action) {
	// 30% chance of making a mistake
	if e.rng.Float64() > 0.3 {
		return
	}

	logger.Info("Stealth: Simulating human imperfection", "action", act.Type)

	switch act.Type {
	case action.ActionSendMessage:
		e.simulateTypo()
	case action.ActionSendConnection:
		e.simulateMouseMisalignment()
	case action.ActionSearchProfiles:
		e.simulateScrollOvershoot()
	default:
		// Generic hesitation for other actions
		time.Sleep(500 * time.Millisecond)
	}
}

// simulateTypo represents a user making a typing mistake and backspacing to fix it.
func (e *Engine) simulateTypo() {
	logger.Debug("Imperfection: Typed wrong character")
	time.Sleep(200 * time.Millisecond) // Time to realize mistake
	logger.Debug("Imperfection: Backspacing...")
	time.Sleep(300 * time.Millisecond) // Time to press backspace
	logger.Debug("Imperfection: Retyping correct character")
	time.Sleep(200 * time.Millisecond) // Time to retype
}

// simulateMouseMisalignment represents missing a button and re-adjusting the mouse.
func (e *Engine) simulateMouseMisalignment() {
	logger.Debug("Imperfection: Mouse overshoot target")
	time.Sleep(400 * time.Millisecond) // Overshoot movement
	logger.Debug("Imperfection: Pausing to correct aim")
	time.Sleep(300 * time.Millisecond) // Reaction time
	logger.Debug("Imperfection: Moving to correct target")
	time.Sleep(400 * time.Millisecond) // Correction movement
}

// simulateScrollOvershoot represents scrolling past the desired content and scrolling back up.
func (e *Engine) simulateScrollOvershoot() {
	logger.Debug("Imperfection: Scrolled too far")
	time.Sleep(600 * time.Millisecond) // Realization pause
	logger.Debug("Imperfection: Scrolling back up")
	time.Sleep(800 * time.Millisecond) // Correction scroll
}
