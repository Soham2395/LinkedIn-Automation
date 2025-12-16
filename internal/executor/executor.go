package executor

import (
	"errors"
	"fmt"
	"linkedin-automation/internal/action"
	"linkedin-automation/internal/logger"
	"linkedin-automation/internal/risk"
	"linkedin-automation/internal/stealth"
)

type Executor struct {
	riskEngine    *risk.Engine
	stealthEngine *stealth.Engine
}

func NewExecutor(riskEngine *risk.Engine, stealthEngine *stealth.Engine) *Executor {
	return &Executor{
		riskEngine:    riskEngine,
		stealthEngine: stealthEngine,
	}
}

func (e *Executor) Execute(act action.Action, fn func() error) error {
	if err := e.riskEngine.Apply(act); err != nil {
		if errors.Is(err, risk.ErrDailyRiskExceeded) {
			logger.Error("Executor: Daily risk limit exceeded, action blocked",
				"action", act.Type,
				"target", act.Target,
				"current_risk", e.riskEngine.Score(),
				"max_daily", e.riskEngine.MaxDaily(),
			)
			return fmt.Errorf("daily limit reached: %w", err)
		}
		logger.Error("Executor: Risk assessment failed",
			"action", act.Type,
			"error", err,
		)
		return fmt.Errorf("risk assessment failed: %w", err)
	}

	e.stealthEngine.Before(act)
	e.stealthEngine.ApplyImperfection(act)

	if err := fn(); err != nil {
		logger.Error("Executor: Action execution failed",
			"action", act.Type,
			"target", act.Target,
			"error", err,
		)
		return err
	}

	e.stealthEngine.After(act)

	logger.Info("Executor: Action completed successfully",
		"action", act.Type,
		"target", act.Target,
		"reason", act.Reason,
		"risk_score", e.riskEngine.Score(),
		"risk_remaining", e.riskEngine.Remaining(),
	)

	return nil
}
