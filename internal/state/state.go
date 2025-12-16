package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-rod/rod/lib/proto"
)

type Store interface {
	SaveCookies(cookies []*proto.NetworkCookie) error
	LoadCookies() ([]*proto.NetworkCookieParam, error)
	CookiesExist() bool

	SaveSearchState(state *SearchState) error
	LoadSearchState() (*SearchState, error)

	IsProfileProcessed(profileURL string) bool
	MarkProfileProcessed(profileURL string) error
}

type SearchState struct {
	LastPage          int      `json:"last_page"`
	ProcessedProfiles []string `json:"processed_profiles"`
}

type FileStore struct {
	dir            string
	processedCache map[string]bool
}

func NewFileStore(dir string) *FileStore {
	os.MkdirAll(dir, 0755)
	store := &FileStore{
		dir:            dir,
		processedCache: make(map[string]bool),
	}
	store.loadProcessedCache()
	return store
}

func (s *FileStore) cookiePath() string {
	return filepath.Join(s.dir, "cookies.json")
}

func (s *FileStore) searchStatePath() string {
	return filepath.Join(s.dir, "search_state.json")
}

func (s *FileStore) CookiesExist() bool {
	_, err := os.Stat(s.cookiePath())
	return err == nil
}

func (s *FileStore) SaveCookies(cookies []*proto.NetworkCookie) error {
	data, err := json.Marshal(cookies)
	if err != nil {
		return err
	}
	return os.WriteFile(s.cookiePath(), data, 0600)
}

func (s *FileStore) LoadCookies() ([]*proto.NetworkCookieParam, error) {
	data, err := os.ReadFile(s.cookiePath())
	if err != nil {
		return nil, err
	}

	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}

	params := make([]*proto.NetworkCookieParam, len(cookies))
	for i, c := range cookies {
		params[i] = &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
		}
	}

	return params, nil
}

func (s *FileStore) SaveSearchState(state *SearchState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.searchStatePath(), data, 0600)
}

func (s *FileStore) LoadSearchState() (*SearchState, error) {
	data, err := os.ReadFile(s.searchStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &SearchState{LastPage: 0, ProcessedProfiles: []string{}}, nil
		}
		return nil, err
	}

	var state SearchState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *FileStore) loadProcessedCache() {
	state, err := s.LoadSearchState()
	if err != nil {
		return
	}
	for _, url := range state.ProcessedProfiles {
		s.processedCache[url] = true
	}
}

func (s *FileStore) IsProfileProcessed(profileURL string) bool {
	return s.processedCache[profileURL]
}

func (s *FileStore) MarkProfileProcessed(profileURL string) error {
	s.processedCache[profileURL] = true

	state, err := s.LoadSearchState()
	if err != nil {
		state = &SearchState{}
	}

	state.ProcessedProfiles = append(state.ProcessedProfiles, profileURL)
	return s.SaveSearchState(state)
}
