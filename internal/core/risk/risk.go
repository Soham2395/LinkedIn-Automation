package risk

import (
	"fmt"
	"linkedin-automation/internal/core/action"
	"linkedin-automation/internal/core/logging"
	"time"
)

type Assessor interface {
	Assess(act action.Action) error
}

type BasicAssessor struct {
}

func NewBasicAssessor() *BasicAssessor {
	return &BasicAssessor{}
}

func (r *BasicAssessor) Assess(act action.Action) error {
	logging.Info("Risk Assessment: Evaluating action", "action", act.Name())

	now := time.Now()
	if now.Hour() >= 2 && now.Hour() < 5 {
		return fmt.Errorf("risk assessment failed: unsafe time window (2AM-5AM)")
	}

	logging.Info("Risk Assessment: Action approved", "action", act.Name())
	return nil
}
