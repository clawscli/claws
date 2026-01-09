package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
)

const (
	DefaultMaxSessions = 100
	sessionDir         = "chat/sessions"
	currentSessionFile = "chat/current.json"
)

type Session struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	Context   *Context  `json:"context,omitempty"`
}

type ContextMode string

const (
	ContextModeSingle ContextMode = "single"
	ContextModeList   ContextMode = "list"
	ContextModeDiff   ContextMode = "diff"
)

type ResourceRef struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Region  string `json:"region,omitempty"`
	Profile string `json:"profile,omitempty"`
	Cluster string `json:"cluster,omitempty"`
}

type Context struct {
	Service      string      `json:"service,omitempty"`
	ResourceType string      `json:"resource_type,omitempty"`
	UserRegions  []string    `json:"user_regions,omitempty"`
	UserProfiles []string    `json:"user_profiles,omitempty"`
	Mode         ContextMode `json:"mode,omitempty"`

	ResourceID      string `json:"resource_id,omitempty"`
	ResourceName    string `json:"resource_name,omitempty"`
	ResourceRegion  string `json:"resource_region,omitempty"`
	ResourceProfile string `json:"resource_profile,omitempty"`
	Cluster         string `json:"cluster,omitempty"`
	LogGroup        string `json:"log_group,omitempty"`

	ResourceCount int             `json:"resource_count,omitempty"`
	FilterText    string          `json:"filter_text,omitempty"`
	Toggles       map[string]bool `json:"toggles,omitempty"`

	DiffLeft  *ResourceRef `json:"diff_left,omitempty"`
	DiffRight *ResourceRef `json:"diff_right,omitempty"`
}

type SessionManager struct {
	maxSessions int
	saveEnabled bool
	currentID   string
}

func NewSessionManager(maxSessions int, saveEnabled bool) *SessionManager {
	if maxSessions <= 0 {
		maxSessions = DefaultMaxSessions
	}
	return &SessionManager{
		maxSessions: maxSessions,
		saveEnabled: saveEnabled,
	}
}

func (m *SessionManager) sessionsDir() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionDir), nil
}

func (m *SessionManager) currentPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, currentSessionFile), nil
}

func (m *SessionManager) NewSession(ctx *Context) (*Session, error) {
	session := &Session{
		ID:        generateSessionID(),
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []Message{},
		Context:   ctx,
	}

	m.currentID = session.ID
	return session, nil
}

func (m *SessionManager) CurrentSession() (*Session, error) {
	if m.currentID == "" {
		path, err := m.currentPath()
		if err != nil {
			return nil, err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}

		var current struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &current); err != nil {
			return nil, err
		}
		m.currentID = current.ID
	}

	if m.currentID == "" {
		return nil, nil
	}

	return m.LoadSession(m.currentID)
}

func (m *SessionManager) LoadSession(id string) (*Session, error) {
	dir, err := m.sessionsDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (m *SessionManager) SaveMessages(session *Session) error {
	session.UpdatedAt = time.Now()
	return m.saveSession(session)
}

func (m *SessionManager) AddMessage(session *Session, msg Message) error {
	session.Messages = append(session.Messages, msg)
	err := m.SaveMessages(session)
	if err != nil {
		return err
	}
	if len(session.Messages) == 1 {
		// Check if pruning is needed before loading all sessions
		shouldPrune, checkErr := m.shouldPrune()
		if checkErr != nil {
			log.Debug("failed to check prune status", "error", checkErr)
		} else if shouldPrune {
			if pruneErr := m.pruneOldSessions(); pruneErr != nil {
				log.Debug("failed to prune old sessions", "error", pruneErr)
			}
		}
	}
	return nil
}

func (m *SessionManager) shouldPrune() (bool, error) {
	dir, err := m.sessionsDir()
	if err != nil {
		return false, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Count only .json files
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}

	return count > m.maxSessions, nil
}

func (m *SessionManager) ListSessions() ([]Session, error) {
	dir, err := m.sessionsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		session, err := m.LoadSession(id)
		if err != nil {
			log.Debug("failed to load session", "id", id, "error", err)
			continue
		}
		sessions = append(sessions, *session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

func (m *SessionManager) saveSession(session *Session) error {
	if !m.saveEnabled {
		return nil
	}

	dir, err := m.sessionsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	return m.saveCurrentID(session.ID)
}

func (m *SessionManager) saveCurrentID(id string) error {
	path, err := m.currentPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: id})

	return os.WriteFile(path, data, 0600)
}

func (m *SessionManager) pruneOldSessions() error {
	dir, err := m.sessionsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect .json files with modification times
	type sessionFile struct {
		name    string
		modTime time.Time
	}
	var files []sessionFile
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, sessionFile{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	if len(files) <= m.maxSessions {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Delete oldest sessions
	deleteCount := len(files) - m.maxSessions
	for i := 0; i < deleteCount; i++ {
		_ = os.Remove(filepath.Join(dir, files[i].name))
	}

	return nil
}

func generateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), uuid.New().String()[:8])
}
