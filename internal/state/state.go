package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

type Store interface {
	SaveCookies(cookies []*proto.NetworkCookie) error
	LoadCookies() ([]*proto.NetworkCookieParam, error)
	CookiesExist() bool

	IsProfileProcessed(profileURL string) bool
	MarkProfileProcessed(profileURL string) error

	IsMessageSent(profileURL string) bool
	MarkMessageSent(profileURL string) error

	SaveSearchState(state *SearchState) error
	LoadSearchState() (*SearchState, error)

	GetDailyStats() DailyStats
	IncrementConnections() error
	IncrementMessages() error

	ResetDailyIfNeeded() error
}

type SearchState struct {
	LastPage          int      `json:"last_page"`
	ProcessedProfiles []string `json:"processed_profiles"`
}

type PersistentState struct {
	LastPage          int        `json:"last_page"`
	ProcessedProfiles []string   `json:"processed_profiles"`
	SentMessages      []string   `json:"sent_messages"`
	DailyStats        DailyStats `json:"daily_stats"`
	LastActivity      time.Time  `json:"last_activity"`
}

type DailyStats struct {
	ConnectionsSent int    `json:"connections_sent"`
	MessagesSent    int    `json:"messages_sent"`
	Date            string `json:"date"`
}

type FileStore struct {
	mu             sync.RWMutex
	dir            string
	processedCache map[string]bool
	messageCache   map[string]bool
	stats          DailyStats
	lastPage       int
}

func NewFileStore(dir string) *FileStore {
	os.MkdirAll(dir, 0755)
	store := &FileStore{
		dir:            dir,
		processedCache: make(map[string]bool),
		messageCache:   make(map[string]bool),
		stats:          DailyStats{Date: today()},
	}
	store.loadState()
	return store
}

func (s *FileStore) cookiePath() string {
	return filepath.Join(s.dir, "cookies.json")
}

func (s *FileStore) statePath() string {
	return filepath.Join(s.dir, "state.json")
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

func (s *FileStore) loadState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.statePath())
	if err != nil {
		return
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	for _, url := range state.ProcessedProfiles {
		s.processedCache[url] = true
	}
	for _, url := range state.SentMessages {
		s.messageCache[url] = true
	}
	s.stats = state.DailyStats
	s.lastPage = state.LastPage

	if s.stats.Date != today() {
		s.stats = DailyStats{Date: today()}
	}
}

func (s *FileStore) saveState() error {
	state := PersistentState{
		LastPage:          s.lastPage,
		ProcessedProfiles: s.mapKeys(s.processedCache),
		SentMessages:      s.mapKeys(s.messageCache),
		DailyStats:        s.stats,
		LastActivity:      time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath(), data, 0600)
}

func (s *FileStore) mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *FileStore) IsProfileProcessed(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processedCache[profileURL]
}

func (s *FileStore) MarkProfileProcessed(profileURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processedCache[profileURL] = true
	return s.saveState()
}

func (s *FileStore) IsMessageSent(profileURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messageCache[profileURL]
}

func (s *FileStore) MarkMessageSent(profileURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageCache[profileURL] = true
	return s.saveState()
}

func (s *FileStore) GetDailyStats() DailyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *FileStore) IncrementConnections() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetDailyIfNeeded()
	s.stats.ConnectionsSent++
	return s.saveState()
}

func (s *FileStore) IncrementMessages() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetDailyIfNeeded()
	s.stats.MessagesSent++
	return s.saveState()
}

func (s *FileStore) ResetDailyIfNeeded() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resetDailyIfNeeded() {
		return s.saveState()
	}
	return nil
}

func (s *FileStore) resetDailyIfNeeded() bool {
	now := today()
	if s.stats.Date != now {
		s.stats = DailyStats{
			Date:            now,
			ConnectionsSent: 0,
			MessagesSent:    0,
		}
		return true
	}
	return false
}

func (s *FileStore) SaveSearchState(state *SearchState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastPage = state.LastPage
	for _, url := range state.ProcessedProfiles {
		s.processedCache[url] = true
	}

	return s.saveState()
}

func (s *FileStore) LoadSearchState() (*SearchState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &SearchState{
		LastPage:          s.lastPage,
		ProcessedProfiles: s.mapKeys(s.processedCache),
	}, nil
}

func today() string {
	return time.Now().Format("2006-01-02")
}
