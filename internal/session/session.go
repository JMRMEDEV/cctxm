package session

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const SessionsDir = "sessions"
const StateFile = "state.json"

type State struct {
	ActiveSession string `json:"active_session"`
	Workspace     string `json:"workspace"`
}

type Meta struct {
	ID          string    `json:"id"`
	Task        string    `json:"task"`
	Keywords    []string  `json:"keywords"`
	FilterMode  string    `json:"filter_mode"`
	CreatedAt   time.Time `json:"created_at"`
	CommandCount int      `json:"command_count"`
}

type CommandEntry struct {
	Number    int       `json:"number"`
	Command   string    `json:"command"`
	ExitCode  int       `json:"exit_code"`
	Duration  string    `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
	RawLog    string    `json:"raw_log"`
	Filtered  string    `json:"filtered_log"`
}

type Manager struct {
	cctxmDir string
}

func NewManager(cctxmDir string) *Manager {
	return &Manager{cctxmDir: cctxmDir}
}

func (m *Manager) sessionsDir() string {
	return filepath.Join(m.cctxmDir, SessionsDir)
}

func (m *Manager) statePath() string {
	return filepath.Join(m.cctxmDir, StateFile)
}

func (m *Manager) sessionDir(id string) string {
	return filepath.Join(m.sessionsDir(), id)
}

// Start creates a new session and sets it as active.
func (m *Manager) Start(description string) (*Meta, error) {
	id := fmt.Sprintf("s_%s_%04d", time.Now().Format("20060102_150405"), rand.Intn(10000))
	dir := m.sessionDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session dir: %w", err)
	}

	meta := &Meta{
		ID:         id,
		Task:       description,
		FilterMode: "normal",
		CreatedAt:  time.Now(),
	}

	if err := m.saveMeta(id, meta); err != nil {
		return nil, err
	}
	if err := m.saveCommands(id, []CommandEntry{}); err != nil {
		return nil, err
	}
	if err := m.SetActive(id); err != nil {
		return nil, err
	}
	return meta, nil
}

// List returns all sessions sorted by creation time (newest first).
func (m *Manager) List() ([]Meta, error) {
	entries, err := os.ReadDir(m.sessionsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Meta
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "s_") {
			continue
		}
		meta, err := m.LoadMeta(e.Name())
		if err != nil {
			continue
		}
		sessions = append(sessions, *meta)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})
	return sessions, nil
}

// Active returns the current active session ID from state.json.
func (m *Manager) Active() (string, error) {
	state, err := m.loadState()
	if err != nil {
		return "", nil
	}
	return state.ActiveSession, nil
}

// SetActive sets the active session in state.json.
func (m *Manager) SetActive(id string) error {
	state, _ := m.loadState()
	if state == nil {
		state = &State{}
	}
	state.ActiveSession = id
	return m.saveState(state)
}

// LoadMeta reads the meta.json for a session.
func (m *Manager) LoadMeta(id string) (*Meta, error) {
	path := filepath.Join(m.sessionDir(id), "meta.json")
	return readJSON[Meta](path)
}

// Restore reactivates a session.
func (m *Manager) Restore(id string) (*Meta, error) {
	meta, err := m.LoadMeta(id)
	if err != nil {
		return nil, fmt.Errorf("session '%s' not found: %w", id, err)
	}
	if err := m.SetActive(id); err != nil {
		return nil, err
	}
	return meta, nil
}

// Clean removes sessions older than the given number of days.
// If days <= 0, removes all sessions.
func (m *Manager) Clean(days int) (int, error) {
	sessions, err := m.List()
	if err != nil {
		return 0, err
	}

	active, _ := m.Active()
	cutoff := time.Now().AddDate(0, 0, -days)
	removed := 0

	for _, s := range sessions {
		if s.ID == active {
			continue
		}
		if days <= 0 || s.CreatedAt.Before(cutoff) {
			os.RemoveAll(m.sessionDir(s.ID))
			removed++
		}
	}
	return removed, nil
}

// LogCommand appends a command entry to the active session.
func (m *Manager) LogCommand(id string, entry CommandEntry) error {
	commands, err := m.LoadCommands(id)
	if err != nil {
		commands = []CommandEntry{}
	}

	entry.Number = len(commands) + 1
	entry.RawLog = fmt.Sprintf("%03d-%s.raw.log", entry.Number, sanitizeLabel(entry.Command))
	entry.Filtered = fmt.Sprintf("%03d-%s.filtered.log", entry.Number, sanitizeLabel(entry.Command))
	commands = append(commands, entry)

	if err := m.saveCommands(id, commands); err != nil {
		return err
	}

	// Update command count in meta
	meta, err := m.LoadMeta(id)
	if err == nil {
		meta.CommandCount = len(commands)
		m.saveMeta(id, meta)
	}
	return nil
}

// LoadCommands reads commands.json for a session.
func (m *Manager) LoadCommands(id string) ([]CommandEntry, error) {
	path := filepath.Join(m.sessionDir(id), "commands.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var commands []CommandEntry
	return commands, json.Unmarshal(data, &commands)
}

// SessionDir returns the directory path for a session (for writing log files).
func (m *Manager) SessionDir(id string) string {
	return m.sessionDir(id)
}

// UpdateMeta saves updated meta for a session.
func (m *Manager) UpdateMeta(id string, meta *Meta) error {
	return m.saveMeta(id, meta)
}

// --- internal helpers ---

func (m *Manager) saveMeta(id string, meta *Meta) error {
	path := filepath.Join(m.sessionDir(id), "meta.json")
	return writeJSON(path, meta)
}

func (m *Manager) saveCommands(id string, commands []CommandEntry) error {
	path := filepath.Join(m.sessionDir(id), "commands.json")
	return writeJSON(path, commands)
}

func (m *Manager) loadState() (*State, error) {
	return readJSON[State](m.statePath())
}

func (m *Manager) saveState(state *State) error {
	return writeJSON(m.statePath(), state)
}

func readJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v T
	return &v, json.Unmarshal(data, &v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func sanitizeLabel(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "cmd"
	}
	label := parts[0]
	if len(parts) > 1 {
		label += "-" + parts[1]
	}
	label = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, label)
	if len(label) > 30 {
		label = label[:30]
	}
	return label
}
