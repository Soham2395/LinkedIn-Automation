package search

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
	"linkedin-automation/internal/logger"
	"linkedin-automation/internal/state"

	"github.com/go-rod/rod"
)

type Criteria struct {
	Keywords string
	JobTitle string
	Company  string
	Location string
	MaxPages int
}

type Profile struct {
	URL      string
	Name     string
	Headline string
	Location string
}

const linkedInSearchBase = "https://www.linkedin.com/search/results/people/"

var nextButtonSelectors = []string{
	"button[aria-label='Next']",
	".artdeco-pagination__button--next",
	"button.artdeco-pagination__button--next",
}

func SearchProfiles(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
	criteria Criteria,
) ([]Profile, error) {
	if criteria.MaxPages <= 0 {
		criteria.MaxPages = 5
	}

	searchState, err := store.LoadSearchState()
	if err != nil {
		searchState = &state.SearchState{LastPage: 0}
	}

	startPage := searchState.LastPage
	if startPage > 0 {
		startPage++
	}

	var profiles []Profile

	for page := startPage; page < criteria.MaxPages; page++ {
		pageProfiles, hasNext, err := searchPage(ctx, exec, b, store, criteria, page)
		if err != nil {
			return profiles, err
		}

		profiles = append(profiles, pageProfiles...)

		searchState.LastPage = page
		store.SaveSearchState(searchState)

		if !hasNext || len(pageProfiles) == 0 {
			break
		}

		if page < criteria.MaxPages-1 {
			if err := paginateToNext(ctx, exec, b); err != nil {
				break
			}
		}
	}

	return profiles, nil
}

func searchPage(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
	store state.Store,
	criteria Criteria,
	page int,
) ([]Profile, bool, error) {
	var profiles []Profile
	var hasNext bool

	searchURL := buildSearchURL(criteria, page)

	act := action.Action{
		Type:       action.ActionSearchProfiles,
		Target:     searchURL,
		Reason:     fmt.Sprintf("Searching page %d", page+1),
		RiskWeight: 0.5,
	}

	err := exec.Execute(act, func() error {
		if err := b.Navigate(searchURL); err != nil {
			return err
		}

		if err := b.WaitLoad(); err != nil {
			return err
		}

		// Wait for dynamic content to load
		time.Sleep(3 * time.Second)
		
		// Check if we got redirected (auth wall, CAPTCHA, etc.)
		currentURL := b.Page().MustInfo().URL
		if strings.Contains(currentURL, "authwall") || 
		   strings.Contains(currentURL, "login") ||
		   strings.Contains(currentURL, "checkpoint") {
			logger.Error("Redirected to auth/checkpoint", "url", currentURL)
			return fmt.Errorf("authentication or verification required")
		}

		parsedProfiles := parseProfilesFromPage(b, store)
		profiles = parsedProfiles

		hasNext = detectNextPage(b)

		return nil
	})

	return profiles, hasNext, err
}

func paginateToNext(
	ctx context.Context,
	exec *executor.Executor,
	b *browser.Browser,
) error {
	act := action.Action{
		Type:       action.ActionPaginateSearch,
		Target:     "next_page",
		Reason:     "Navigating to next search results page",
		RiskWeight: 0.2,
	}

	return exec.Execute(act, func() error {
		for _, selector := range nextButtonSelectors {
			if b.HasElement(selector) {
				return b.Click(selector)
			}
		}
		return fmt.Errorf("next button not found")
	})
}

func buildSearchURL(criteria Criteria, page int) string {
	params := url.Values{}

	if criteria.Keywords != "" {
		params.Set("keywords", criteria.Keywords)
	}
	if criteria.JobTitle != "" {
		params.Set("titleFreeText", criteria.JobTitle)
	}
	if criteria.Company != "" {
		params.Set("company", criteria.Company)
	}
	if criteria.Location != "" {
		params.Set("geoUrn", criteria.Location)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page+1))
	}

	return linkedInSearchBase + "?" + params.Encode()
}

func parseProfilesFromPage(b *browser.Browser, store state.Store) []Profile {
	var profiles []Profile

	page := b.Page()
	if page == nil {
		logger.Error("Page is nil")
		return profiles
	}

	// STRATEGY: LinkedIn changed their HTML structure
	// Instead of looking for containers, find all profile links directly
	// then extract information from their parent/sibling elements
	
	// First, try the traditional container approach
	containerSelectors := []string{
		"li.reusable-search__result-container",
		"ul.reusable-search__entity-result-list > li",
		"div.search-results-container li",
		"li[class*='reusable-search']",
	}

	var containers rod.Elements
	var err error
	
	for _, selector := range containerSelectors {
		containers, err = page.Elements(selector)
		if err == nil && len(containers) > 0 {
			logger.Info("Found profile containers", "selector", selector, "count", len(containers))
			break
		}
		fmt.Printf("[DEBUG] Tried selector '%s': %d elements\n", selector, len(containers))
	}

	// If containers found, use the traditional approach
	if len(containers) > 0 {
		for i, container := range containers {
			profile := extractProfileFromContainer(container, i)
			
			if profile.URL == "" {
				continue
			}

			if store.IsProfileProcessed(profile.URL) {
				logger.Info("Skipping already processed profile", "url", profile.URL)
				continue
			}

			// Deduplicate
			isDuplicate := false
			for _, p := range profiles {
				if p.URL == profile.URL {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				profiles = append(profiles, profile)
				logger.Info("Added profile", "name", profile.Name, "url", profile.URL)
			}
		}
	} else {
		// FALLBACK: Extract directly from profile links
		logger.Info("Using fallback method: extracting profiles directly from links")
		profiles = extractProfilesFromLinks(page, store)
	}

	logger.Info("Successfully parsed profiles", "total", len(profiles))
	
	if len(profiles) == 0 {
		logger.Error("Parsed 0 profiles despite finding links")
		dumpDebugInfo(page)
	}

	return profiles
}

// New function to extract profiles directly from links (fallback method)
func extractProfilesFromLinks(page *rod.Page, store state.Store) []Profile {
	var profiles []Profile
	
	// Find all links to LinkedIn profiles
	allLinks, err := page.Elements("a[href*='/in/']")
	if err != nil || len(allLinks) == 0 {
		logger.Error("No profile links found on page")
		return profiles
	}
	
	logger.Info("Found profile links", "count", len(allLinks))
	
	seen := make(map[string]bool)
	
	for i, link := range allLinks {
		href, err := link.Attribute("href")
		if err != nil || href == nil {
			continue
		}
		
		profileURL := *href
		
		// Clean URL
		if idx := strings.Index(profileURL, "?"); idx != -1 {
			profileURL = profileURL[:idx]
		}
		if !strings.HasPrefix(profileURL, "http") {
			profileURL = "https://www.linkedin.com" + profileURL
		}
		
		// Skip non-profile links or duplicates
		if !strings.Contains(profileURL, "/in/") || seen[profileURL] {
			continue
		}
		
		// Skip already processed
		if store.IsProfileProcessed(profileURL) {
			continue
		}
		
		// Get the text from the link
		text, err := link.Text()
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		
		// LinkedIn profile links often have the name in them
		// Skip links with very long text (likely contain entire profile card)
		// or very short text (likely icons/buttons)
		if len(text) < 5 || len(text) > 100 {
			continue
		}
		
		// Skip common non-profile links
		skipTexts := []string{"Follow", "Connect", "Message", "More", "Premium"}
		skip := false
		for _, skipText := range skipTexts {
			if text == skipText {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		
		profile := Profile{
			URL:  profileURL,
			Name: text,
		}
		
		// Try to extract additional info from parent element
		parent, err := link.Parent()
		if err == nil && parent != nil {
			// Try to find headline/subtitle near the link
			grandparent, err := parent.Parent()
			if err == nil && grandparent != nil {
				// Look for subtitle elements in the grandparent
				if subtitle, err := grandparent.Element(".t-14"); err == nil {
					if subtext, err := subtitle.Text(); err == nil {
						profile.Headline = strings.TrimSpace(subtext)
					}
				}
			}
		}
		
		profiles = append(profiles, profile)
		seen[profileURL] = true
		
		logger.Info("Extracted profile from link", "index", i, "name", profile.Name, "url", profile.URL)
		
		// Limit to reasonable number per page
		if len(profiles) >= 10 {
			break
		}
	}
	
	return profiles
}

func extractProfileFromContainer(container *rod.Element, index int) Profile {
	profile := Profile{}

	// Modern LinkedIn structure (2024+):
	// The profile link is typically in a span with class "entity-result__title-text"
	// or directly as an anchor with href containing "/in/"
	
	linkSelectors := []string{
		// Primary selectors for modern LinkedIn
		"span.entity-result__title-text a.app-aware-link",
		"a.app-aware-link[href*='/in/']",
		"div.entity-result__content a[href*='/in/']",
		// Fallback selectors
		"a[href*='/in/']",
		".entity-result__title-text a",
	}

	var profileLink *rod.Element
	
	for _, selector := range linkSelectors {
		if link, err := container.Element(selector); err == nil {
			profileLink = link
			break
		}
	}

	if profileLink == nil {
		// Try to get HTML to debug
		html, _ := container.HTML()
		if len(html) > 500 {
			html = html[:500]
		}
		fmt.Printf("[DEBUG] Container %d: No profile link found. HTML preview: %s...\n", index, html)
		return profile
	}

	// Extract URL
	href, err := profileLink.Attribute("href")
	if err != nil || href == nil {
		return profile
	}

	profileURL := *href
	if !strings.Contains(profileURL, "/in/") {
		return profile
	}

	// Clean URL (remove query parameters)
	if idx := strings.Index(profileURL, "?"); idx != -1 {
		profileURL = profileURL[:idx]
	}
	if !strings.HasPrefix(profileURL, "http") {
		profileURL = "https://www.linkedin.com" + profileURL
	}
	profile.URL = profileURL

	// Extract name from the link or nearby span
	// Modern LinkedIn puts the name in a nested span with aria-hidden="true"
	nameSelectors := []string{
		"span[aria-hidden='true']",  // Most common
		"span.entity-result__title-text span[aria-hidden='true']",
	}

	for _, selector := range nameSelectors {
		if nameEl, err := container.Element(selector); err == nil {
			if name, err := nameEl.Text(); err == nil && strings.TrimSpace(name) != "" {
				profile.Name = strings.TrimSpace(name)
				break
			}
		}
	}

	// Fallback: get text from link itself
	if profile.Name == "" {
		if text, err := profileLink.Text(); err == nil {
			profile.Name = strings.TrimSpace(text)
		}
	}

	// Extract headline (job title) - usually in primary-subtitle
	headlineSelectors := []string{
		".entity-result__primary-subtitle",
		"div.entity-result__primary-subtitle",
		"div[class*='primary-subtitle']",
	}
	
	for _, selector := range headlineSelectors {
		if headlineEl, err := container.Element(selector); err == nil {
			if headline, err := headlineEl.Text(); err == nil {
				profile.Headline = strings.TrimSpace(headline)
				break
			}
		}
	}

	// Extract location - usually in secondary-subtitle
	locationSelectors := []string{
		".entity-result__secondary-subtitle",
		"div.entity-result__secondary-subtitle",
		"div[class*='secondary-subtitle']",
	}
	
	for _, selector := range locationSelectors {
		if locEl, err := container.Element(selector); err == nil {
			if location, err := locEl.Text(); err == nil {
				profile.Location = strings.TrimSpace(location)
				break
			}
		}
	}

	if profile.Name != "" {
		fmt.Printf("[DEBUG] Extracted profile %d: name='%s', headline='%s', location='%s'\n", 
			index, profile.Name, profile.Headline, profile.Location)
	}

	return profile
}

func dumpDebugInfo(page *rod.Page) {
	fmt.Println("\n[DEBUG] ===== DUMPING DEBUG INFORMATION =====")
	
	// Save HTML
	html, _ := page.HTML()
	_ = os.WriteFile("debug_search.html", []byte(html), 0644)
	fmt.Println("[DEBUG] Saved HTML to debug_search.html")
	
	// Save screenshot
	screenshot, err := page.Screenshot(false, nil)
	if err == nil {
		_ = os.WriteFile("debug_search.png", screenshot, 0644)
		fmt.Println("[DEBUG] Saved screenshot to debug_search.png")
	}
	
	// Log current URL
	fmt.Printf("[DEBUG] Current URL: %s\n", page.MustInfo().URL)
	
	// Try to find ANY links to profiles
	allLinks, _ := page.Elements("a[href*='/in/']")
	fmt.Printf("[DEBUG] Found %d links containing '/in/' on the page\n", len(allLinks))
	
	if len(allLinks) > 0 {
		fmt.Println("[DEBUG] Sample links found:")
		for i, link := range allLinks {
			if i >= 3 {
				break
			}
			href, _ := link.Attribute("href")
			text, _ := link.Text()
			if href != nil {
				fmt.Printf("  [%d] href='%s' text='%s'\n", i, *href, text)
			}
		}
	}
	
	// Check for list items
	listItems, _ := page.Elements("li")
	fmt.Printf("[DEBUG] Found %d <li> elements total on page\n", len(listItems))
	
	// Check for common class patterns
	fmt.Println("[DEBUG] Checking for common class patterns:")
	patterns := []string{"reusable-search", "entity-result", "search-result"}
	for _, pattern := range patterns {
		elements, _ := page.Elements(fmt.Sprintf("[class*='%s']", pattern))
		fmt.Printf("  - Elements with class containing '%s': %d\n", pattern, len(elements))
	}
	
	fmt.Println("[DEBUG] ===== END DEBUG INFO =====\n")
}

func detectNextPage(b *browser.Browser) bool {
	for _, selector := range nextButtonSelectors {
		if b.HasElement(selector) {
			return true
		}
	}
	return false
}