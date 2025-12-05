package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager handles conversation history persistence
type Manager struct {
	filePath   string
	mu         sync.RWMutex
	history    *History
	current    *Session
	maxSessions int
}

// NewManager creates a new history manager
func NewManager(filePath string, maxSessions int) *Manager {
	return &Manager{
		filePath:    filePath,
		history:     &History{Sessions: []Session{}},
		maxSessions: maxSessions,
	}
}

// Load loads history from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty history
		m.history = &History{Sessions: []Session{}}
		m.startNewSession()
		return nil
	}

	// Read file
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &m.history); err != nil {
		// Corrupted file - backup and start fresh
		backupPath := m.filePath + ".backup"
		os.Rename(m.filePath, backupPath)
		m.history = &History{Sessions: []Session{}}
	}

	// Start a new session
	m.startNewSession()

	return nil
}

// startNewSession creates a new session (must be called with lock held)
func (m *Manager) startNewSession() {
	now := time.Now()
	m.current = &Session{
		ID:        uuid.New().String(),
		StartedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}
	m.history.Sessions = append(m.history.Sessions, *m.current)
}

// AddMessage adds a message to the current session
func (m *Manager) AddMessage(msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		m.startNewSession()
	}

	// Set timestamp if not already set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Add to current session
	m.current.Messages = append(m.current.Messages, msg)
	m.current.UpdatedAt = time.Now()

	// Update in history
	for i := range m.history.Sessions {
		if m.history.Sessions[i].ID == m.current.ID {
			m.history.Sessions[i] = *m.current
			break
		}
	}

	// Save to disk
	return m.saveUnlocked()
}

// GetRecentMessages returns the last N messages from the current session
func (m *Manager) GetRecentMessages(limit int) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil || len(m.current.Messages) == 0 {
		return []Message{}
	}

	messages := m.current.Messages
	if len(messages) <= limit {
		return messages
	}

	return messages[len(messages)-limit:]
}

// Save persists the history to disk
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveUnlocked()
}

// saveUnlocked saves without acquiring the lock (must be called with lock held)
func (m *Manager) saveUnlocked() error {
	// Prune old sessions if needed
	if len(m.history.Sessions) > m.maxSessions {
		m.history.Sessions = m.history.Sessions[len(m.history.Sessions)-m.maxSessions:]
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Write to temp file
	tempPath := m.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetCurrentSession returns the current session
func (m *Manager) GetCurrentSession() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}
