package session

import (
	"encoding/json"
	"fmt"

	"lark_ai/common/code"
	"lark_ai/controller"
	"lark_ai/model"
	"lark_ai/service/session"

	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sse"
)

type (
	GetUserSessionsResponse struct {
		controller.Response
		Sessions []model.SessionInfo `json:"sessions,omitempty"`
	}
	CreateSessionAndSendMessageRequest struct {
		UserQuestion string `json:"question" binding:"required"`   // User question
		ModelType    string `json:"model_type" binding:"required"` // Model type
	}

	CreateSessionAndSendMessageResponse struct {
		AiInformation string `json:"information,omitempty"` // AI response
		SessionID     string `json:"session_id,omitempty"`  // Current session ID
		controller.Response
	}

	ChatSendRequest struct {
		UserQuestion string `json:"question" binding:"required"`             // User question
		ModelType    string `json:"model_type" binding:"required"`           // Model type
		SessionID    string `json:"session_id,omitempty" binding:"required"` // Current session ID
	}

	ChatSendResponse struct {
		AiInformation string `json:"information,omitempty"` // AI response
		controller.Response
	}

	ChatHistoryRequest struct {
		SessionID string `json:"session_id,omitempty" binding:"required"` // Current session ID
	}
	ChatHistoryResponse struct {
		History []model.History `json:"history"`
		controller.Response
	}
)

func GetUserSessionsByUserName(ctx context.Context, c *app.RequestContext) {
	res := new(GetUserSessionsResponse)
	username := c.GetString("username") // From JWT middleware

	userSessions, err := session.GetUserSessionsByUserName(username)
	if err != nil {
		c.JSON(200, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Sessions = userSessions
	c.JSON(200, res)
}

func CreateSessionAndSendMessage(ctx context.Context, c *app.RequestContext) {
	req := new(CreateSessionAndSendMessageRequest)
	res := new(CreateSessionAndSendMessageResponse)
	username := c.GetString("username") // From JWT middleware
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}
	// Internally creates session, sends message, and returns AI response and current session
	session_id, aiInformation, code_ := session.CreateSessionAndSendMessage(username, req.UserQuestion, req.ModelType)

	if code_ != code.CodeSuccess {
		c.JSON(200, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	res.SessionID = session_id
	c.JSON(200, res)
}

func CreateStreamSessionAndSendMessage(ctx context.Context, c *app.RequestContext) {
	req := new(CreateSessionAndSendMessageRequest)
	username := c.GetString("username") // From JWT middleware
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, map[string]any{"error": "Invalid parameters"})
		return
	}

	// Create session first
	sessionID, code_ := session.CreateStreamSessionOnly(username, req.UserQuestion)
	if code_ != code.CodeSuccess {
		c.JSON(200, map[string]any{"message": "Failed to create session"})
		return
	}

	// Enable stream response
	c.SetStatusCode(200)
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	s := sse.NewStream(c)

	// Send sessionId to frontend first
	initMsg := fmt.Sprintf("{\"sessionId\": \"%s\"}", sessionID)
	s.Publish(&sse.Event{
		Data: []byte(initMsg),
	})

	sendFunc := func(msg string) error {
		if msg == "[DONE]" {
			err := s.Publish(&sse.Event{Data: []byte("[DONE]")})
			return err
		}
		b, _ := json.Marshal(map[string]string{"content": msg})
		err := s.Publish(&sse.Event{
			Data: b,
		})
		return err
	}

	session.StreamMessageToExistingSession(username, sessionID, req.UserQuestion, req.ModelType, sendFunc)
}

func ChatSend(ctx context.Context, c *app.RequestContext) {
	req := new(ChatSendRequest)
	res := new(ChatSendResponse)
	username := c.GetString("username") // From JWT middleware
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}
	// Send message and return AI response
	aiInformation, code_ := session.ChatSend(username, req.SessionID, req.UserQuestion, req.ModelType)

	if code_ != code.CodeSuccess {
		c.JSON(200, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	c.JSON(200, res)
}

func ChatStreamSend(ctx context.Context, c *app.RequestContext) {
	req := new(ChatSendRequest)
	username := c.GetString("username") // From JWT middleware
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, map[string]any{"error": "Invalid parameters"})
		return
	}

	c.SetStatusCode(200)
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	s := sse.NewStream(c)

	sendFunc := func(msg string) error {
		if msg == "[DONE]" {
			err := s.Publish(&sse.Event{Data: []byte("[DONE]")})
			c.Flush()
			return err
		}
		b, _ := json.Marshal(map[string]string{"content": msg})
		err := s.Publish(&sse.Event{
			Data: b,
		})
		c.Flush()
		return err
	}

	session.ChatStreamSend(username, req.SessionID, req.UserQuestion, req.ModelType, sendFunc)
}

func ChatHistory(ctx context.Context, c *app.RequestContext) {
	req := new(ChatHistoryRequest)
	res := new(ChatHistoryResponse)
	username := c.GetString("username") // From JWT middleware
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}
	history, code_ := session.GetChatHistory(username, req.SessionID)
	if code_ != code.CodeSuccess {
		c.JSON(200, res.CodeOf(code_))
		return
	}

	res.Success()
	res.History = history
	c.JSON(200, res)
}
