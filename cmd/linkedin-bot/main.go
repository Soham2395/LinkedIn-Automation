package main

import (
	"context"
	"os"

	"linkedin-automation/internal/auth"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/logger"
	"linkedin-automation/internal/messaging"
	"linkedin-automation/internal/profile"
	"linkedin-automation/internal/risk"
	"linkedin-automation/internal/search"
	"linkedin-automation/internal/state"
	"linkedin-automation/internal/stealth"

	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	logger.Info("LinkedIn Automation Bot - Starting")

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, relying on environment variables")
	}
	username := os.Getenv("LINKEDIN_USERNAME")
	password := os.Getenv("LINKEDIN_PASSWORD")
	if username == "" || password == "" {
		logger.Error("Missing credentials. Please set LINKEDIN_USERNAME and LINKEDIN_PASSWORD.")
		os.Exit(1)
	}

	stealthProfile := stealth.ProfileNormal
	riskEngine := risk.NewEngine(stealthProfile.MaxDailyRisk)
	stealthEngine := stealth.NewEngine(stealthProfile)
	exec := executor.NewExecutor(riskEngine, stealthEngine)
	store := state.NewFileStore("./data")
	b, err := browser.New(false)
	if err != nil {
		logger.Error("Failed to initialize browser", "error", err)
		os.Exit(1)
	}
	defer b.Close()

	logger.Info("Step 1: Authenticating...")
	if err := auth.Authenticate(ctx, exec, b, store); err != nil {
		logger.Error("Authentication failed", "error", err)
		os.Exit(1)
	}
	logger.Info("Authentication successful")

	logger.Info("Step 2: Searching for profiles...")
	criteria := search.Criteria{
		Keywords: "Software Engineer",
		// Location: "San Francisco Bay Area", // geoUrn requires an ID, not text
		MaxPages: 1,
	}

	profiles, err := search.SearchProfiles(ctx, exec, b, store, criteria)
	if err != nil {
		logger.Error("Search failed", "error", err)
	}
	logger.Info("Found profiles", "count", len(profiles))

	logger.Info("Step 3: Visiting and Connecting...")
	processedCount := 0
	for _, p := range profiles {
		if processedCount >= 1 {
			break
		}

		if store.IsProfileProcessed(p.URL) {
			continue
		}

		logger.Info("Processing profile", "name", p.Name, "url", p.URL)

		note := "Hi {{FirstName}}, I noticed you work at {{Company}} as a {{Role}}. Would love to connect!"

		if err := profile.VisitAndConnect(ctx, exec, b, store, p, note); err != nil {
			logger.Error("Failed to process profile", "profile", p.Name, "error", err)
			continue
		}
		processedCount++
	}
	logger.Info("Step 4: Sending Follow-Ups...")
	followUpTemplate := messaging.MessageTemplate{
		Content: "Hi {{FirstName}}, thanks for connecting! I'd love to hear more about your work at {{Company}}.",
	}

	if err := messaging.SendFollowUps(ctx, exec, b, store, followUpTemplate); err != nil {
		logger.Error("Failed to send follow-ups", "error", err)
	}

	stats := store.GetDailyStats()
	logger.Info("LinkedIn Automation Bot - Finished",
		"final_risk_score", riskEngine.Score(),
		"connections_sent", stats.ConnectionsSent,
		"messages_sent", stats.MessagesSent,
	)
}
