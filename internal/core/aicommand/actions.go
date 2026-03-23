package aicommand

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ActionType identifies the kind of action the AI determined.
type ActionType string

const (
	// Read-only actions (execute immediately).
	ActionNavigate      ActionType = "navigate"
	ActionSearchMovie   ActionType = "search_movie"
	ActionQueryLibrary  ActionType = "query_library"
	ActionSearchRelease ActionType = "search_releases"
	ActionExplain       ActionType = "explain"
	ActionFallback      ActionType = "fallback"

	// State-modifying actions (require confirmation).
	ActionAutoSearch ActionType = "auto_search"
	ActionRunTask    ActionType = "run_task"
)

// RequiresConfirmation returns true for state-modifying actions.
func (a ActionType) RequiresConfirmation() bool {
	switch a { //nolint:exhaustive // only state-modifying actions need confirmation
	case ActionAutoSearch, ActionRunTask:
		return true
	}
	return false
}

// CommandResponse is the structured response returned to the frontend.
type CommandResponse struct {
	Action              ActionType     `json:"action"`
	Params              map[string]any `json:"params,omitempty"`
	Result              map[string]any `json:"result,omitempty"`
	Explanation         string         `json:"explanation"`
	RequiresConfirm     bool           `json:"requires_confirmation,omitempty"`
	PendingActionID     string         `json:"pending_action_id,omitempty"`
	ConfirmationMessage string         `json:"confirmation_message,omitempty"`
}

// PendingAction is a state-modifying action awaiting user confirmation.
type PendingAction struct {
	ID        string
	Action    ActionType
	Params    map[string]any
	CreatedAt time.Time
}

// PendingStore holds actions awaiting confirmation. Entries expire after 5 minutes.
type PendingStore struct {
	mu      sync.Mutex
	actions map[string]*PendingAction
}

// NewPendingStore creates an empty pending action store.
func NewPendingStore() *PendingStore {
	return &PendingStore{actions: make(map[string]*PendingAction)}
}

// Add stores a pending action and returns its ID.
func (s *PendingStore) Add(action ActionType, params map[string]any) string {
	id := randomID()
	s.mu.Lock()
	// Prune expired entries while we're here.
	cutoff := time.Now().Add(-5 * time.Minute)
	for k, v := range s.actions {
		if v.CreatedAt.Before(cutoff) {
			delete(s.actions, k)
		}
	}
	s.actions[id] = &PendingAction{
		ID:        id,
		Action:    action,
		Params:    params,
		CreatedAt: time.Now(),
	}
	s.mu.Unlock()
	return id
}

// Take retrieves and removes a pending action. Returns nil if not found or expired.
func (s *PendingStore) Take(id string) *PendingAction {
	s.mu.Lock()
	defer s.mu.Unlock()
	pa, ok := s.actions[id]
	if !ok {
		return nil
	}
	delete(s.actions, id)
	if time.Since(pa.CreatedAt) > 5*time.Minute {
		return nil
	}
	return pa
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
