package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/clawscli/claws/internal/config"
)

const (
	DefaultMaxSessions = 10
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

type Context struct {
	Service        string   `json:"service,omitempty"`
	ResourceType   string   `json:"resource_type,omitempty"`
	ResourceID     string   `json:"resource_id,omitempty"`
	ResourceName   string   `json:"resource_name,omitempty"`
	ResourceRegion string   `json:"resource_region,omitempty"`
	Cluster        string   `json:"cluster,omitempty"` // ECS cluster name (for tasks/services)
	LogGroup       string   `json:"log_group,omitempty"`
	Regions        []string `json:"regions,omitempty"`
}

type SessionManager struct {
	maxSessions int
	currentID   string
}

func NewSessionManager(maxSessions int) *SessionManager {
	if maxSessions <= 0 {
		maxSessions = DefaultMaxSessions
	}
	return &SessionManager{
		maxSessions: maxSessions,
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

	if err := m.saveSession(session); err != nil {
		return nil, err
	}

	m.currentID = session.ID

	if err := m.pruneOldSessions(); err != nil {
		return session, nil
	}

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
	return m.SaveMessages(session)
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
	dir, err := m.sessionsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: id})

	return os.WriteFile(path, data, 0644)
}

func (m *SessionManager) pruneOldSessions() error {
	sessions, err := m.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) <= m.maxSessions {
		return nil
	}

	dir, err := m.sessionsDir()
	if err != nil {
		return err
	}

	for i := m.maxSessions; i < len(sessions); i++ {
		path := filepath.Join(dir, sessions[i].ID+".json")
		_ = os.Remove(path)
	}

	return nil
}

func generateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%d", now.Format("2006-01-02"), now.UnixNano()%1000000)
}
