package session

import (
	"context"
	"log"
	"lark_ai/common/ai_agent"
	"lark_ai/common/code"
	"lark_ai/dao/session"
	"lark_ai/model"

	"github.com/google/uuid"
)

var ctx = context.Background()

func GetUserSessionsByUserName(username string) ([]model.SessionInfo, error) {
	// Get all session IDs for user

	manager := ai_agent.GetGlobalManager()
	Sessions := manager.GetUserSessions(username)

	var SessionInfos []model.SessionInfo

	for _, session := range Sessions {
		SessionInfos = append(SessionInfos, model.SessionInfo{
			SessionID: session,
			Title:     session, // Temporarily use sessionID as title, can be changed during refactor
		})
	}

	return SessionInfos, nil
}

func CreateSessionAndSendMessage(username string, userQuestion string, modelType string) (string, string, code.Code) {
	// 1: Create a new session
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: username,
		Title:    userQuestion, // Can set title as needed, temporarily using user's first question
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	// 2: Get AIAgent and manage messages through it
	manager := ai_agent.GetGlobalManager()
	config := map[string]any{
		"apiKey":   "your-api-key", // TODO: Get from config
		"username": username,       // Used for RAG model to get user document
	}
	helper, err := manager.GetOrCreateAIAgent(username, createdSession.ID, modelType, config)
	if err != nil {
		log.Println("CreateSessionAndSendMessage GetOrCreateAIAgent error:", err)
		return "", "", code.AIModelFail
	}

	// 3: Generate AI reply
	aiResponse, err_ := helper.GenerateResponse(username, ctx, userQuestion)
	if err_ != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", err_)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func CreateStreamSessionOnly(username string, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: username,
		Title:    userQuestion,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(username string, sessionID string, userQuestion string, modelType string, sendFunc func(msg string) error) code.Code {
	manager := ai_agent.GetGlobalManager()
	config := map[string]any{
		"apiKey":   "your-api-key", // TODO: Get from config
		"username": username,       // Used for RAG model to get user document
	}
	helper, err := manager.GetOrCreateAIAgent(username, sessionID, modelType, config)
	if err != nil {
		log.Println("StreamMessageToExistingSession GetOrCreateAIAgent error:", err)
		return code.AIModelFail
	}

	cb := func(msg string) {
		log.Printf("[SSE] Sending chunk: %s (len=%d)\n", msg, len(msg))
		if err := sendFunc(msg); err != nil {
			log.Println("[SSE] Write error:", err)
		}
		log.Println("[SSE] Flushed")
	}

	_, err_ := helper.StreamResponse(username, ctx, cb, userQuestion)
	if err_ != nil {
		log.Println("StreamMessageToExistingSession StreamResponse error:", err_)
		return code.AIModelFail
	}

	if err := sendFunc("[DONE]"); err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.AIModelFail
	}

	return code.CodeSuccess
}

func CreateStreamSessionAndSendMessage(username string, userQuestion string, modelType string, sendFunc func(msg string) error) (string, code.Code) {

	sessionID, code_ := CreateStreamSessionOnly(username, userQuestion)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	code_ = StreamMessageToExistingSession(username, sessionID, userQuestion, modelType, sendFunc)
	if code_ != code.CodeSuccess {
		return sessionID, code_
	}

	return sessionID, code.CodeSuccess
}

func ChatSend(username string, sessionID string, userQuestion string, modelType string) (string, code.Code) {
	// 1: Get AIAgent
	manager := ai_agent.GetGlobalManager()
	config := map[string]any{
		"username": username, // Used for RAG model to get user document (used if user selects RAG model)
	}
	helper, err := manager.GetOrCreateAIAgent(username, sessionID, modelType, config)
	if err != nil {
		log.Println("ChatSend GetOrCreateAIAgent error:", err)
		return "", code.AIModelFail
	}

	// 2: Generate AI reply
	aiResponse, err_ := helper.GenerateResponse(username, ctx, userQuestion)
	if err_ != nil {
		log.Println("ChatSend GenerateResponse error:", err_)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(username string, sessionID string) ([]model.History, code.Code) {
	// Get message history from AIAgent
	manager := ai_agent.GetGlobalManager()
	helper, exists := manager.GetAIAgent(username, sessionID)
	if !exists {
		return nil, code.CodeServerBusy
	}

	messages := helper.GetMessages()
	history := make([]model.History, 0, len(messages))

	// Convert messages to history format (judge user/AI message by order or content)
	for i, msg := range messages {
		isUser := i%2 == 0
		history = append(history, model.History{
			IsUser:  isUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

func ChatStreamSend(username string, sessionID string, userQuestion string, modelType string, sendFunc func(msg string) error) code.Code {

	return StreamMessageToExistingSession(username, sessionID, userQuestion, modelType, sendFunc)
}
