package ai_agent

import (
	"context"
	"lark_ai/common/kafka"
	"lark_ai/model"
	"lark_ai/utils"
	"sync"
)

// AIAgent AI assistant struct, contains message history and AI model
type AIAgent struct {
	model    AIModel
	messages []*model.Message
	mu       sync.RWMutex
	// One session binds to one AIAgent
	SessionID string
	saveFunc  func(*model.Message) (*model.Message, error)
}

// NewAIAgent creates new AIAgent instance
func NewAIAgent(model_ AIModel, SessionID string) *AIAgent {
	return &AIAgent{
		model:    model_,
		messages: make([]*model.Message, 0),
		// Asynchronously push to message queue
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			data := kafka.GenerateMessageMQParam(msg.SessionID, msg.Content, msg.UserName, msg.IsUser)
			err := kafka.GlobalKafkaClient.Publish(context.Background(), data)
			return msg, err
		},
		SessionID: SessionID,
	}
}

// addMessage adds message to memory and calls custom storage function
func (a *AIAgent) AddMessage(Content string, UserName string, IsUser bool, Save bool) {
	userMsg := model.Message{
		SessionID: a.SessionID,
		Content:   Content,
		UserName:  UserName,
		IsUser:    IsUser,
	}
	a.messages = append(a.messages, &userMsg)
	if Save {
		a.saveFunc(&userMsg)
	}
}

// SaveMessage saves message to database (avoids circular dependency via callback)
// Supports multiple strategies (sync/async) by passing external save function
func (a *AIAgent) SetSaveFunc(saveFunc func(*model.Message) (*model.Message, error)) {
	a.saveFunc = saveFunc
}

// GetMessages gets all message history
func (a *AIAgent) GetMessages() []*model.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*model.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// Sync generation
func (a *AIAgent) GenerateResponse(username string, ctx context.Context, userQuestion string) (*model.Message, error) {

	// Call storage function
	a.AddMessage(userQuestion, username, true, true)

	a.mu.RLock()
	// Convert model.Message to schema.Message
	messages := utils.ConvertToSchemaMessages(a.messages)
	a.mu.RUnlock()

	// Call model to generate reply
	schemaMsg, err := a.model.GenerateResponse(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Convert schema.Message to model.Message
	modelMsg := utils.ConvertToModelMessage(a.SessionID, username, schemaMsg)

	// Call storage function
	a.AddMessage(modelMsg.Content, username, false, true)

	return modelMsg, nil
}

// Stream generation
func (a *AIAgent) StreamResponse(username string, ctx context.Context, cb StreamCallback, userQuestion string) (*model.Message, error) {

	// Call storage function
	a.AddMessage(userQuestion, username, true, true)

	a.mu.RLock()
	messages := utils.ConvertToSchemaMessages(a.messages)
	a.mu.RUnlock()

	content, err := a.model.StreamResponse(ctx, messages, cb)
	if err != nil {
		return nil, err
	}
	// Convert to model.Message
	modelMsg := &model.Message{
		SessionID: a.SessionID,
		UserName:  username,
		Content:   content,
		IsUser:    false,
	}

	// Call storage function
	a.AddMessage(modelMsg.Content, username, false, true)

	return modelMsg, nil
}

// GetModelType gets model type
func (a *AIAgent) GetModelType() string {
	return a.model.GetModelType()
}
