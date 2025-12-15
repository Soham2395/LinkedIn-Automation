package executor

import (
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
	riskScore := int(act.RiskWeight * 10)
	if err := e.riskEngine.Apply(riskScore); err != nil {
		logger.Error("Executor: Risk threshold exceeded, aborting action",
			"action", act.Type,
			"target", act.Target,
			"risk_added", riskScore,
			"current_risk", e.riskEngine.Score(),
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
	)

	return nil
}
