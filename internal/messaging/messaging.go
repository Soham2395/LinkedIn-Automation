package messaging

import (
	"context"
	"errors"
	"strings"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/search"
	"linkedin-automation/internal/state"

	"github.com/go-rod/rod"
)

const linkedInConnections = "https://www.linkedin.com/mynetwork/invite-connect/connections/"

type MessageTemplate struct {
	Content string
}

func SendFollowUps(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
	template MessageTemplate,
) error {
	newConnections, err := detectAcceptedConnections(ctx, exec, b, store)
	if err != nil {
		return err
	}

	for _, p := range newConnections {
		if store.IsMessageSent(p.URL) {
			continue
		}

		if err := sendMessage(ctx, exec, b, p, template); err != nil {
			return err
		}

		if err := store.MarkMessageSent(p.URL); err != nil {
			return err
		}
	}

	return nil
}

func detectAcceptedConnections(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
) ([]search.Profile, error) {
	var connections []search.Profile

	act := action.Action{
		Type:       action.ActionDetectAcceptance,
		Target:     linkedInConnections,
		Reason:     "Checking for new connections",
		RiskWeight: 0.2,
	}

	err := exec.Execute(act, func() error {
		if err := b.Navigate(linkedInConnections); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		items, err := b.Page().Elements(".mn-connection-card")
		if err != nil {
			return nil 
		}

		for _, item := range items {
			p := extractProfileFromCard(item)
			if p.URL != "" {
				if store.IsProfileProcessed(p.URL) && !store.IsMessageSent(p.URL) {
					connections = append(connections, p)
				}
			}
		}
		return nil
	})

	return connections, err
}

func sendMessage(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	p search.Profile,
	template MessageTemplate,
) error {
	act := action.Action{
		Type:       action.ActionSendMessage,
		Target:     p.Name,
		Reason:     "Sending follow-up message",
		RiskWeight: 0.8,
	}

	return exec.Execute(act, func() error {
		if err := b.Navigate(p.URL); err != nil {
			return err
		}
		if err := b.WaitLoad(); err != nil {
			return err
		}
		if err := clickMessageButton(b); err != nil {
			return err
		}
		content := personalize(template.Content, p)
		if err := b.TypeInto(".msg-form__contenteditable", content); err != nil {
			return err
		}
		return clickSendButton(b)
	})
}

func extractProfileFromCard(el *rod.Element) search.Profile {
	link, err := el.Element("a.mn-connection-card__link")
	if err != nil {
		return search.Profile{}
	}

	href, err := link.Attribute("href")
	if err != nil || href == nil {
		return search.Profile{}
	}

	nameEl, err := el.Element(".mn-connection-card__name")
	name := ""
	if err == nil {
		name, _ = nameEl.Text()
	}

	url := *href
	if !strings.HasPrefix(url, "http") {
		url = "https://www.linkedin.com" + url
	}

	return search.Profile{
		URL:  url,
		Name: strings.TrimSpace(name),
	}
}

func clickMessageButton(b *browser.Browser) error {
	selectors := []string{
		"button.entry-point-btn--message",
		"button[aria-label^='Message']",
		"a.message-anywhere-button",
	}

	for _, s := range selectors {
		if b.HasElement(s) {
			return b.Click(s)
		}
	}
	return errors.New("message button not found")
}

func clickSendButton(b *browser.Browser) error {
	selectors := []string{
		"button[type='submit'].msg-form__send-button",
		"button.msg-form__send-button",
	}

	for _, s := range selectors {
		if b.HasElement(s) {
			return b.Click(s)
		}
	}
	return errors.New("send button not found")
}

func personalize(content string, p search.Profile) string {
	firstName := strings.Split(p.Name, " ")[0]
	text := strings.ReplaceAll(content, "{{FirstName}}", firstName)
	text = strings.ReplaceAll(text, "{{Company}}", "your company")
	text = strings.ReplaceAll(text, "{{Role}}", "your role")
	return text
}
