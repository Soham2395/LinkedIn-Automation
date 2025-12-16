package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/search"
	"linkedin-automation/internal/state"
)

var (
	ErrConnectButtonNotFound = errors.New("connect button not found")
	ErrAlreadyConnected      = errors.New("already connected or pending")
)

var connectButtonSelectors = []string{
	"button.pvs-profile-actions__action",
	"button[aria-label*='Invite'][aria-label*='to connect']",
	"button[aria-label^='Connect']",
	"button:contains('Connect')", 
}

var sendButtonSelectors = []string{
	"button[aria-label='Send now']",
	"button.artdeco-button--primary",
}

var addNoteButtonSelectors = []string{
	"button[aria-label='Add a note']",
	"button.artdeco-button--secondary",
}

func VisitAndConnect(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
	p search.Profile,
	noteTemplate string,
) error {
	if store.IsProfileProcessed(p.URL) {
		return nil
	}

	if err := visitProfile(ctx, exec, b, p); err != nil {
		return err
	}
	if err := sendConnectionRequest(ctx, exec, b, p, noteTemplate); err != nil {
		if errors.Is(err, ErrConnectButtonNotFound) || errors.Is(err, ErrAlreadyConnected) {
			store.MarkProfileProcessed(p.URL)
			return nil
		}
		return err
	}

	return store.MarkProfileProcessed(p.URL)
}

func visitProfile(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	p search.Profile,
) error {
	act := action.Action{
		Type:       action.ActionVisitProfile,
		Target:     p.URL,
		Reason:     "Reviewing profile before connecting",
		RiskWeight: 0.2,
	}

	return exec.Execute(act, func() error {
		if err := b.Navigate(p.URL); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		return nil
	})
}

func sendConnectionRequest(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	p search.Profile,
	noteTemplate string,
) error {
	act := action.Action{
		Type:       action.ActionSendConnection,
		Target:     p.Name,
		Reason:     "Sending connection request",
		RiskWeight: 1.0,
	}

	return exec.Execute(act, func() error {
		if isAlreadyConnected(b) {
			return ErrAlreadyConnected
		}

		if err := clickConnect(b); err != nil {
			if err := tryMoreMenu(b); err != nil {
				return ErrConnectButtonNotFound
			}
		}
		if noteTemplate != "" {
			if err := addNote(b, p, noteTemplate); err != nil {
				return err
			}
		}
		return clickSend(b)
	})
}

func isAlreadyConnected(b *browser.Browser) bool {
	if b.HasElement("button[aria-label^='Message']") {
		return !b.HasElement("button[aria-label^='Connect']")
	}
	if b.HasElement("button:contains('Pending')") {
		return true
	}
	return false
}

func clickConnect(b *browser.Browser) error {
	for _, selector := range connectButtonSelectors {
		if strings.Contains(selector, ":contains") {
			continue
		}
		if b.HasElement(selector) {
			return b.Click(selector)
		}
	}

	return ErrConnectButtonNotFound
}

func tryMoreMenu(b *browser.Browser) error {
	moreBtn := "button[aria-label='More actions']"
	if !b.HasElement(moreBtn) {
		return errors.New("more button not found")
	}
	if err := b.Click(moreBtn); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

	return ErrConnectButtonNotFound
}

func addNote(b *browser.Browser, p search.Profile, template string) error {
	clicked := false
	for _, selector := range addNoteButtonSelectors {
		if b.HasElement(selector) {
			if err := b.Click(selector); err == nil {
				clicked = true
				break
			}
		}
	}
	if !clicked {
		return nil
	}
	note := personalize(template, p)
	return b.TypeInto("textarea[name='message']", note)
}

func clickSend(b *browser.Browser) error {
	for _, selector := range sendButtonSelectors {
		if b.HasElement(selector) {
			return b.Click(selector)
		}
	}
	return nil
}

func personalize(template string, p search.Profile) string {
	firstName := strings.Split(p.Name, " ")[0]
	note := strings.ReplaceAll(template, "{{FirstName}}", firstName)
	note = strings.ReplaceAll(note, "{{Company}}", "your company") 
	note = strings.ReplaceAll(note, "{{Role}}", "your role")   
	return note
}
