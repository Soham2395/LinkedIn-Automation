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
}

type FileStore struct {
	dir string
}

func NewFileStore(dir string) *FileStore {
	os.MkdirAll(dir, 0755)
	return &FileStore{dir: dir}
}

func (s *FileStore) cookiePath() string {
	return filepath.Join(s.dir, "cookies.json")
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
