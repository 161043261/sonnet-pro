package ai_agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"lark_ai/common/rag"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type StreamCallback func(msg string)

// AIModel defines AI model interface
type AIModel interface {
	GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error)
	GetModelType() string
}

// =================== Ollama Implementation ===================

// OllamaModel Ollama model implementation
type OllamaModel struct {
	llm model.ToolCallingChatModel
}

func NewOllamaModel(ctx context.Context, baseURL, modelName string) (*OllamaModel, error) {
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama model failed: %v", err)
	}
	return &OllamaModel{llm: llm}, nil
}

func (o *OllamaModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ollama generate failed: %v", err)
	}
	return resp, nil
}

func (o *OllamaModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ollama stream failed: %v", err)
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ollama stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content) // Aggregate
			cb(msg.Content)                   // Call cb function in real-time for frontend push
		}
	}
	return fullResp.String(), nil // Return full content for subsequent storage
}

func (o *OllamaModel) GetModelType() string { return "ollama" }

// =================== Ollama RAG Implementation ===================

type OllamaRAGModel struct {
	llm      model.ToolCallingChatModel
	username string
}

func NewOllamaRAGModel(ctx context.Context, baseURL, modelName, username string) (*OllamaRAGModel, error) {
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama rag model failed: %v", err)
	}
	return &OllamaRAGModel{
		llm:      llm,
		username: username,
	}, nil
}

func (o *OllamaRAGModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	ragQuery, err := rag.NewRAGQuery(ctx, o.username)
	if err != nil {
		log.Printf("Failed to create RAG query (user may not have uploaded file): %v", err)
		resp, err := o.llm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ollama rag generate failed: %v", err)
		}
		return resp, nil
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content

	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		log.Printf("Failed to retrieve documents: %v", err)
		resp, err := o.llm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ollama rag generate failed: %v", err)
		}
		return resp, nil
	}

	ragPrompt := rag.BuildRAGPrompt(query, docs)
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{
		Role:    schema.User,
		Content: ragPrompt,
	}

	resp, err := o.llm.Generate(ctx, ragMessages)
	if err != nil {
		return nil, fmt.Errorf("ollama rag generate failed: %v", err)
	}
	return resp, nil
}

func (o *OllamaRAGModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	ragMessages, err := o.buildRAGMessages(ctx, messages)
	if err != nil {
		log.Printf("RAG preparation failed, using original messages: %v", err)
		ragMessages = messages
	}

	stream, err := o.llm.Stream(ctx, ragMessages)
	if err != nil {
		return "", fmt.Errorf("ollama rag stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ollama rag stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (o *OllamaRAGModel) buildRAGMessages(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
	ragQuery, err := rag.NewRAGQuery(ctx, o.username)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}
	query := messages[len(messages)-1].Content
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	ragPrompt := rag.BuildRAGPrompt(query, docs)
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{
		Role:    schema.User,
		Content: ragPrompt,
	}
	return ragMessages, nil
}

func (o *OllamaRAGModel) GetModelType() string { return "ollama-rag" }
