package accounts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusCreating  Status = "creating"
	StatusLoggingIn Status = "logging_in"
	StatusReady     Status = "ready"
	StatusExpired   Status = "expired"
	StatusError     Status = "error"
)

type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	PlanType    string `json:"planType,omitempty"`
	ProfilePath string `json:"profilePath"`
	Status      Status `json:"status"`
	CreatedAt   string `json:"createdAt"`
	LastUsedAt  string `json:"lastUsedAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

type data struct {
	ActiveAccountID string    `json:"activeAccountId,omitempty"`
	Accounts        []Account `json:"accounts"`
}

type Store struct {
	mu       sync.Mutex
	root     string
	filePath string
	data     data
}

func NewStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	root := filepath.Join(base, "Mux")
	store := &Store{root: root, filePath: filepath.Join(root, "accounts.json")}
	if err := os.MkdirAll(filepath.Join(root, "profiles"), 0o700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	contents, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		s.data = data{Accounts: []Account{}}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read account data: %w", err)
	}
	if err := json.Unmarshal(contents, &s.data); err != nil {
		return fmt.Errorf("decode account data: %w", err)
	}
	return nil
}

func (s *Store) saveLocked() error {
	contents, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode account data: %w", err)
	}
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, contents, 0o600); err != nil {
		return fmt.Errorf("write account data: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("replace account data: %w", err)
	}
	return nil
}

func (s *Store) List() []Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := make([]Account, len(s.data.Accounts))
	copy(accounts, s.data.Accounts)
	return accounts
}

func (s *Store) Active() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.ActiveAccountID
}

func (s *Store) Create(name string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.NewString()
	if name == "" {
		name = id
	}
	profilePath := filepath.Join(s.root, "profiles", id)
	if err := os.MkdirAll(profilePath, 0o700); err != nil {
		return Account{}, fmt.Errorf("create account profile: %w", err)
	}
	account := Account{ID: id, Name: name, ProfilePath: profilePath, Status: StatusCreating, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.data.Accounts = append(s.data.Accounts, account)
	if s.data.ActiveAccountID == "" {
		s.data.ActiveAccountID = id
	}
	if err := s.saveLocked(); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (s *Store) UpdateProfile(id string, name string, email string, avatarURL string, planType string) error {
	if name == "" {
		return errors.New("account name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts {
		if s.data.Accounts[i].ID == id {
			s.data.Accounts[i].Name = name
			s.data.Accounts[i].Email = email
			s.data.Accounts[i].AvatarURL = avatarURL
			s.data.Accounts[i].PlanType = planType
			return s.saveLocked()
		}
	}
	return errors.New("account not found")
}

func (s *Store) SetStatus(id string, status Status, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts {
		if s.data.Accounts[i].ID == id {
			s.data.Accounts[i].Status = status
			s.data.Accounts[i].Error = message
			return s.saveLocked()
		}
	}
	return errors.New("account not found")
}

func (s *Store) SetActive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts {
		if s.data.Accounts[i].ID == id {
			s.data.ActiveAccountID = id
			s.data.Accounts[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339)
			return s.saveLocked()
		}
	}
	return errors.New("account not found")
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, account := range s.data.Accounts {
		if account.ID == id {
			if err := os.RemoveAll(account.ProfilePath); err != nil {
				return fmt.Errorf("remove account profile: %w", err)
			}
			s.data.Accounts = append(s.data.Accounts[:i], s.data.Accounts[i+1:]...)
			if s.data.ActiveAccountID == id {
				s.data.ActiveAccountID = ""
				if len(s.data.Accounts) > 0 {
					s.data.ActiveAccountID = s.data.Accounts[0].ID
				}
			}
			return s.saveLocked()
		}
	}
	return errors.New("account not found")
}
