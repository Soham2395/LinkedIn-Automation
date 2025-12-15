package main

import (
	"fmt"
	"linkedin-automation/internal/action"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/logger"
	"linkedin-automation/internal/risk"
	"linkedin-automation/internal/stealth"
)

func main() {
	logger.Info("LinkedIn Automation Bot - Starting")

	riskEngine := risk.NewEngine(100)
	stealthProfile := stealth.Profile{
		SpeedMultiplier: 1.0, 
		JitterFactor:    0.2,
	}
	stealthEngine := stealth.NewEngine(stealthProfile)
	exec := executor.NewExecutor(riskEngine, stealthEngine)
	searchAction := action.Action{
		Type:       action.ActionSearchProfiles,
		Target:     "Software Engineers in San Francisco",
		Reason:     "Finding potential candidates for outreach",
		RiskWeight: 0.5, // Medium risk
	}

	err := exec.Execute(searchAction, func() error {
		fmt.Println("[BROWSER] Executing search...")
		return nil
	})

	if err != nil {
		logger.Error("Failed to execute search action", "error", err)
	}

	visitAction := action.Action{
		Type:       action.ActionVisitProfile,
		Target:     "https://linkedin.com/in/john-doe",
		Reason:     "Reviewing candidate background",
		RiskWeight: 0.2,
	}

	err = exec.Execute(visitAction, func() error {
		fmt.Println("[BROWSER] Visiting profile...")
		return nil
	})

	if err != nil {
		logger.Error("Failed to execute visit action", "error", err)
	}

	connectAction := action.Action{
		Type:       action.ActionSendConnection,
		Target:     "john-doe",
		Reason:     "Expanding professional network",
		RiskWeight: 0.8, 
	}

	err = exec.Execute(connectAction, func() error {
		fmt.Println("[BROWSER] Sending connection request...")
		return nil
	})

	if err != nil {
		logger.Error("Failed to execute connect action", "error", err)
	}

	logger.Info("LinkedIn Automation Bot - Finished", "final_risk_score", riskEngine.Score())
}
