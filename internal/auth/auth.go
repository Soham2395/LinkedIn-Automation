package auth

import (
	"context"
	"errors"
	"os"
	"strings"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/state"
)

var (
	ErrCheckpointDetected = errors.New("security checkpoint detected")
	ErrLoginFailed        = errors.New("login failed")
	ErrMissingCredentials = errors.New("missing credentials")
)

const (
	linkedInHome  = "https://www.linkedin.com"
	linkedInLogin = "https://www.linkedin.com/login"
	linkedInFeed  = "https://www.linkedin.com/feed"
)

var checkpointPatterns = []string{
	"/checkpoint/",
	"/challenge/",
	"/security-verification",
	"/authwall",
}

var checkpointTextIndicators = []string{
	"verify it's you",
	"security verification",
	"unusual activity",
	"confirm your identity",
}

func Authenticate(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
) error {
	if store.CookiesExist() {
		if err := tryRestoreSession(ctx, exec, b, store); err == nil {
			return nil
		}
	}

	return performLogin(ctx, exec, b, store)
}

func tryRestoreSession(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
) error {
	act := action.Action{
		Type:       action.ActionRestoreSession,
		Target:     linkedInHome,
		Reason:     "Attempting to restore existing session",
		RiskWeight: 0.2,
	}

	return exec.Execute(act, func() error {
		cookies, err := store.LoadCookies()
		if err != nil {
			return err
		}

		if err := b.Navigate(linkedInHome); err != nil {
			return err
		}

		if err := b.SetCookies(cookies); err != nil {
			return err
		}

		if err := b.Navigate(linkedInFeed); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		if err := detectCheckpoint(b); err != nil {
			return err
		}

		if !isLoggedIn(b) {
			return errors.New("session invalid")
		}

		return nil
	})
}

func performLogin(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
) error {
	username := os.Getenv("LINKEDIN_USERNAME")
	password := os.Getenv("LINKEDIN_PASSWORD")

	if username == "" || password == "" {
		return ErrMissingCredentials
	}

	act := action.Action{
		Type:       action.ActionLogin,
		Target:     linkedInLogin,
		Reason:     "Performing fresh login",
		RiskWeight: 1.0,
	}

	return exec.Execute(act, func() error {
		if err := b.Navigate(linkedInLogin); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		if err := b.TypeInto("#username", username); err != nil {
			return err
		}

		if err := b.TypeInto("#password", password); err != nil {
			return err
		}

		if err := b.Click("button[type='submit']"); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		if err := detectCheckpoint(b); err != nil {
			return err
		}

		if !isLoggedIn(b) {
			return ErrLoginFailed
		}

		return persistCookies(b, store)
	})
}

func detectCheckpoint(b *browser.Browser) error {
	url := b.URL()
	for _, pattern := range checkpointPatterns {
		if strings.Contains(url, pattern) {
			return ErrCheckpointDetected
		}
	}

	for _, indicator := range checkpointTextIndicators {
		if b.HasElement("body") {
			text, err := b.TextContent("body")
			if err == nil && strings.Contains(strings.ToLower(text), indicator) {
				return ErrCheckpointDetected
			}
		}
	}

	return nil
}

func isLoggedIn(b *browser.Browser) bool {
	url := b.URL()
	if strings.Contains(url, "/feed") {
		return true
	}
	if b.HasElement(".global-nav__me") {
		return true
	}
	if b.HasElement("[data-test-id='nav-menu']") {
		return true
	}
	return false
}

func persistCookies(b *browser.Browser, store state.Store) error {
	cookies, err := b.GetCookies()
	if err != nil {
		return err
	}
	return store.SaveCookies(cookies)
}
