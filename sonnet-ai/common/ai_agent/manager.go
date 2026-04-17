package ai_agent

import (
	"context"
	"sync"
)

var ctx = context.Background()

// AIAgentManager manages mapping of User-Session-AIAgent
type AIAgentManager struct {
	helpers map[string]map[string]*AIAgent // map[UserAccount(Unique)]map[SessionID]*AIAgent
	mu      sync.RWMutex
}

// NewAIAgentManager creates new manager instance
func NewAIAgentManager() *AIAgentManager {
	return &AIAgentManager{
		helpers: make(map[string]map[string]*AIAgent),
	}
}

// Get or create AIAgent
func (m *AIAgentManager) GetOrCreateAIAgent(username string, sessionID string, modelType string, config map[string]any) (*AIAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get user's session mapping
	userHelpers, exists := m.helpers[username]
	if !exists {
		userHelpers = make(map[string]*AIAgent)
		m.helpers[username] = userHelpers
	}

	// Check if session already exists
	helper, exists := userHelpers[sessionID]
	if exists {
		return helper, nil
	}

	// Create new AIAgent
	factory := GetGlobalFactory()
	helper, err := factory.CreateAIAgent(ctx, modelType, sessionID, config)
	if err != nil {
		return nil, err
	}

	userHelpers[sessionID] = helper
	return helper, nil
}

// Get AIAgent for specific user and session
func (m *AIAgentManager) GetAIAgent(username string, sessionID string) (*AIAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[username]
	if !exists {
		return nil, false
	}

	helper, exists := userHelpers[sessionID]
	return helper, exists
}

// Remove AIAgent for specific user and session
func (m *AIAgentManager) RemoveAIAgent(username string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[username]
	if !exists {
		return
	}

	delete(userHelpers, sessionID)

	// Clean up user mapping if no sessions left
	if len(userHelpers) == 0 {
		delete(m.helpers, username)
	}
}

// Get all session IDs for specific user
func (m *AIAgentManager) GetUserSessions(username string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[username]
	if !exists {
		return []string{}
	}

	sessionIDs := make([]string, 0, len(userHelpers))
	// Extract all keys
	for sessionID := range userHelpers {
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs
}

// Global manager instance
var globalManager *AIAgentManager
var once sync.Once

// GetGlobalManager gets global manager instance
func GetGlobalManager() *AIAgentManager {
	once.Do(func() {
		globalManager = NewAIAgentManager()
	})
	return globalManager
}
