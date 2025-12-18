package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/logger"
	"linkedin-automation/internal/search"
	"linkedin-automation/internal/state"

	"github.com/chromedp/chromedp"
)

var (
	ErrConnectButtonNotFound = errors.New("connect button not found")
	ErrAlreadyConnected      = errors.New("already connected or pending")
)

var connectButtonSelectors = []string{
	"button.pvs-profile-actions__action",
	"button[aria-label*='Invite'][aria-label*='to connect']",
	"button[aria-label^='Connect']",
	"button[aria-label*='Connect with']",
}

var sendButtonSelectors = []string{
	"button[aria-label='Send now']",
	"button[aria-label='Send invitation']",
	"button.artdeco-button--primary",
}

var addNoteButtonSelectors = []string{
	"button[aria-label='Add a note']",
	"button.artdeco-button--secondary",
}

var moreMenuSelectors = []string{
	"button[aria-label='More actions']",
	"button[aria-label^='More']",
	"div[aria-label='More actions'] button",
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
		// DEBUG: Log all buttons on the page
		debugButtons(ctx, p.Name)

		// Check if already connected first
		if isAlreadyConnected(ctx, b) {
			return ErrAlreadyConnected
		}

		// Try direct connect button first
		logger.Info("Attempting direct connect button", "profile", p.Name)
		if err := clickConnect(b); err != nil {
			logger.Info("Direct connect failed, trying More menu", "profile", p.Name, "error", err)
			// If direct connect fails, try the More menu
			if err := tryMoreMenu(ctx, b); err != nil {
				logger.Error("More menu also failed", "profile", p.Name, "error", err)
				return ErrConnectButtonNotFound
			}
		} else {
			logger.Info("Direct connect succeeded", "profile", p.Name)
		}

		// Wait for connection modal to appear
		time.Sleep(1 * time.Second)

		// Add note if template provided
		if noteTemplate != "" {
			if err := addNote(b, p, noteTemplate); err != nil {
				logger.Error("Failed to add note", "error", err)
			}
		}

		// Click Send button
		return clickSend(ctx, b)
	})
}

func debugButtons(ctx context.Context, profileName string) {
	script := `
		(function() {
			const buttons = Array.from(document.querySelectorAll('button'));
			return buttons.map(btn => ({
				text: btn.innerText.trim().substring(0, 50),
				ariaLabel: btn.getAttribute('aria-label') || '',
				className: btn.className.substring(0, 100)
			})).filter(b => b.text || b.ariaLabel);
		})()
	`

	var buttonInfo []map[string]string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &buttonInfo),
	)

	if err == nil && len(buttonInfo) > 0 {
		logger.Info("=== BUTTONS ON PAGE ===", "profile", profileName)
		for i, btn := range buttonInfo {
			if i < 15 { // Limit to first 15 buttons
				logger.Info("Button found",
					"index", i,
					"text", btn["text"],
					"aria-label", btn["ariaLabel"],
					"class", btn["className"],
				)
			}
		}
		logger.Info("======================")
	}
}

func isAlreadyConnected(ctx context.Context, b *browser.Browser) bool {
	// FIRST: Check if Connect button exists - if it does, we're NOT connected
	// This is the most reliable indicator
	connectButtonExists := false
	for _, selector := range connectButtonSelectors {
		if b.HasElement(selector) {
			logger.Info("Found Connect button - user is NOT connected", "selector", selector)
			connectButtonExists = true
			break
		}
	}

	// If we found a Connect button, definitely not connected
	if connectButtonExists {
		return false
	}

	// Use JavaScript for a more thorough check
	script := `
		(function() {
			const buttons = Array.from(document.querySelectorAll('button'));
			let hasConnect = false;
			let hasMessage = false;
			let hasPending = false;

			for (let btn of buttons) {
				const text = btn.innerText.trim().toLowerCase();
				const ariaLabel = (btn.getAttribute('aria-label') || '').toLowerCase();
				
				// Check for connect button
				if (text === 'connect' || ariaLabel.includes('connect') || ariaLabel.includes('invite')) {
					hasConnect = true;
				}
				
				// Check for message button (but only as primary action, not secondary)
				if ((text === 'message' && !ariaLabel.includes('connect')) || 
					(ariaLabel.startsWith('message') && !text.includes('connect'))) {
					hasMessage = true;
				}
				
				// Check for pending
				if (text === 'pending' || ariaLabel.includes('pending')) {
					hasPending = true;
				}
			}

			// If Connect button exists, not connected (even if Message also exists)
			if (hasConnect) {
				return 'not_connected';
			}

			// If Pending exists, invitation already sent
			if (hasPending) {
				return 'pending';
			}

			// If only Message exists (no Connect), already connected
			if (hasMessage) {
				return 'connected';
			}

			return 'unknown';
		})()
	`

	var status string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &status),
	)

	if err == nil {
		logger.Info("Connection status detected", "status", status)
		return status == "connected" || status == "pending"
	}

	// Default: assume not connected to avoid blocking valid connections
	return false
}

func clickConnect(b *browser.Browser) error {
	// Try each connect button selector
	for _, selector := range connectButtonSelectors {
		if b.HasElement(selector) {
			logger.Info("Found connect button with selector", "selector", selector)
			if err := b.Click(selector); err == nil {
				return nil
			}
		}
	}

	return ErrConnectButtonNotFound
}

func tryMoreMenu(ctx context.Context, b *browser.Browser) error {
	// Try to find and click the More/More actions button
	logger.Info("Looking for More button...")

	moreClicked := false
	for _, selector := range moreMenuSelectors {
		if b.HasElement(selector) {
			logger.Info("Found More button", "selector", selector)
			if err := b.Click(selector); err == nil {
				moreClicked = true
				break
			}
		}
	}

	if !moreClicked {
		// Try using JavaScript to find More button
		script := `
			(function() {
				const buttons = Array.from(document.querySelectorAll('button'));
				for (let btn of buttons) {
					const text = btn.innerText.trim();
					const ariaLabel = btn.getAttribute('aria-label') || '';
					if (text === 'More' || ariaLabel.includes('More actions')) {
						btn.click();
						return true;
					}
				}
				return false;
			})()
		`

		var clicked bool
		err := chromedp.Run(ctx,
			chromedp.Evaluate(script, &clicked),
		)

		if err != nil || !clicked {
			return errors.New("more button not found")
		}
		moreClicked = true
	}

	logger.Info("More button clicked, waiting for dropdown...")
	// Wait for dropdown menu to appear
	time.Sleep(1000 * time.Millisecond)

	// Debug: Log dropdown contents
	debugDropdown(ctx)

	// Now look for Connect option in the dropdown menu
	logger.Info("Looking for Connect in dropdown...")
	connectFound := false
	for _, selector := range []string{
		"div[role='menu'] button[aria-label*='Connect']",
		"div.artdeco-dropdown__content button[aria-label*='Connect']",
		"li[role='menuitem'] div[aria-label*='Connect']",
		"div.artdeco-dropdown__item[aria-label*='Connect']",
	} {
		if b.HasElement(selector) {
			logger.Info("Found Connect in dropdown", "selector", selector)
			if err := b.Click(selector); err == nil {
				connectFound = true
				break
			}
		}
	}

	if !connectFound {
		logger.Info("Standard selectors failed, trying JavaScript search...")
		// Try a more generic approach using chromedp
		if err := clickConnectInDropdown(ctx, b); err != nil {
			return errors.New("connect option not found in dropdown")
		}
	}

	logger.Info("Connect option clicked successfully")
	return nil
}

func debugDropdown(ctx context.Context) {
	script := `
		(function() {
			const dropdowns = document.querySelectorAll('div[role="menu"], div.artdeco-dropdown__content, ul[role="menu"]');
			const items = [];
			
			dropdowns.forEach(dropdown => {
				const elements = dropdown.querySelectorAll('li, div[role="menuitem"], a, button');
				elements.forEach(el => {
					const text = el.innerText.trim().substring(0, 50);
					const ariaLabel = el.getAttribute('aria-label') || '';
					if (text || ariaLabel) {
						items.push({ text, ariaLabel });
					}
				});
			});
			
			return items;
		})()
	`

	var items []map[string]string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &items),
	)

	if err == nil && len(items) > 0 {
		logger.Info("=== DROPDOWN ITEMS ===")
		for i, item := range items {
			logger.Info("Dropdown item", "index", i, "text", item["text"], "aria-label", item["ariaLabel"])
		}
		logger.Info("======================")
	} else {
		logger.Info("No dropdown items found or dropdown not visible")
	}
}

func clickConnectInDropdown(ctx context.Context, b *browser.Browser) error {
	// Use chromedp to find and click Connect in dropdown
	script := `
		(function() {
			// Look in dropdown menus
			const dropdowns = document.querySelectorAll('div[role="menu"], div.artdeco-dropdown__content, ul[role="menu"]');
			
			for (let dropdown of dropdowns) {
				const items = dropdown.querySelectorAll('li, div[role="menuitem"], a, button, span');
				
				for (let item of items) {
					const text = item.innerText.trim();
					const ariaLabel = item.getAttribute('aria-label') || '';
					
					if (text === 'Connect' || text.includes('Connect') || ariaLabel.includes('Connect')) {
						// Find the clickable element
						let clickable = item;
						if (item.tagName !== 'BUTTON' && item.tagName !== 'A') {
							clickable = item.querySelector('button') || item.querySelector('a') || item;
						}
						
						clickable.click();
						return true;
					}
				}
			}
			
			return false;
		})()
	`

	var clicked bool
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &clicked),
	)

	if err != nil {
		return fmt.Errorf("failed to execute dropdown script: %w", err)
	}

	if !clicked {
		return errors.New("could not find Connect in dropdown")
	}

	return nil
}

func addNote(b *browser.Browser, p search.Profile, template string) error {
	// Try to click "Add a note" button
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
		// No "Add a note" option available, just continue without note
		return nil
	}

	// Wait for textarea to appear
	time.Sleep(500 * time.Millisecond)

	// Personalize and type the note
	note := personalize(template, p)

	// Try different textarea selectors
	textareaSelectors := []string{
		"textarea[name='message']",
		"textarea[id='custom-message']",
		"textarea.artdeco-text-input",
	}

	for _, selector := range textareaSelectors {
		if b.HasElement(selector) {
			return b.TypeInto(selector, note)
		}
	}

	return errors.New("note textarea not found")
}

func clickSend(ctx context.Context, b *browser.Browser) error {
	// Wait a bit to ensure modal is ready
	time.Sleep(500 * time.Millisecond)

	logger.Info("Looking for Send button...")

	// Try each send button selector
	for _, selector := range sendButtonSelectors {
		if b.HasElement(selector) {
			logger.Info("Found Send button", "selector", selector)
			return b.Click(selector)
		}
	}

	// Try to find Send button using chromedp as fallback
	logger.Info("Standard selectors failed, using JavaScript to find Send button...")

	script := `
		(function() {
			const buttons = document.querySelectorAll('button');
			for (let btn of buttons) {
				const text = btn.textContent.trim();
				const ariaLabel = btn.getAttribute('aria-label') || '';
				if (text === 'Send' || 
					text === 'Send now' || 
					text === 'Send invitation' ||
					text === 'Send without a note' ||
					ariaLabel.includes('Send')) {
					btn.click();
					return true;
				}
			}
			return false;
		})()
	`

	var clicked bool
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &clicked),
	)

	if err != nil {
		return fmt.Errorf("failed to execute send button script: %w", err)
	}

	if !clicked {
		return errors.New("send button not found")
	}

	logger.Info("Send button clicked successfully")
	return nil
}

func personalize(template string, p search.Profile) string {
	firstName := strings.Split(p.Name, " ")[0]
	note := strings.ReplaceAll(template, "{{FirstName}}", firstName)
	note = strings.ReplaceAll(note, "{{Company}}", "your company")
	note = strings.ReplaceAll(note, "{{Role}}", "your role")
	return note
}
