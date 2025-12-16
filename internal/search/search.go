package search

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"linkedin-automation/internal/action"
	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/executor"
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
	URL  string
	Name string
}

const linkedInSearchBase = "https://www.linkedin.com/search/results/people/"

var profileLinkSelectors = []string{
	".entity-result__title-text a",
	".reusable-search__result-container a[href*='/in/']",
	"a.app-aware-link[href*='/in/']",
}

var nextButtonSelectors = []string{
	"button[aria-label='Next']",
	".artdeco-pagination__button--next",
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
		return profiles
	}

	for _, selector := range profileLinkSelectors {
		elements, err := page.Elements(selector)
		if err != nil {
			continue
		}

		for _, el := range elements {
			profile := extractProfile(el)
			if profile.URL == "" {
				continue
			}

			if store.IsProfileProcessed(profile.URL) {
				continue
			}

			profiles = append(profiles, profile)
		}

		if len(profiles) > 0 {
			break
		}
	}

	return profiles
}

func extractProfile(el *rod.Element) Profile {
	href, err := el.Attribute("href")
	if err != nil || href == nil {
		return Profile{}
	}

	profileURL := *href
	if !strings.Contains(profileURL, "/in/") {
		return Profile{}
	}

	if idx := strings.Index(profileURL, "?"); idx != -1 {
		profileURL = profileURL[:idx]
	}

	if !strings.HasPrefix(profileURL, "http") {
		profileURL = "https://www.linkedin.com" + profileURL
	}

	name := ""
	text, err := el.Text()
	if err == nil {
		name = strings.TrimSpace(text)
	}

	return Profile{
		URL:  profileURL,
		Name: name,
	}
}

func detectNextPage(b *browser.Browser) bool {
	for _, selector := range nextButtonSelectors {
		if b.HasElement(selector) {
			return true
		}
	}
	return false
}
