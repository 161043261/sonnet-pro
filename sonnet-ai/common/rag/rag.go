package rag

import (
	"context"
	"fmt"
	"os"
	"lark_ai/common/redis"
	redisPkg "lark_ai/common/redis"
	"lark_ai/config"

	embeddingOllama "github.com/cloudwego/eino-ext/components/embedding/ollama"
	redisIndexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisRetriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"
)

type RAGIndexer struct {
	embedding embedding.Embedder
	indexer   *redisIndexer.Indexer
}

type RAGQuery struct {
	embedding embedding.Embedder
	retriever retriever.Retriever
}

// Build knowledge base index
// Technical: text parsing, chunking, vectorization, vector storage
// Simplified: convert readable documents to AI semantic search format and store
func NewRAGIndexer(filename, embeddingModel string) (*RAGIndexer, error) {

	// Context for initialization flow (timeout/cancel), default background is fine here
	ctx := context.Background()

	// Vector dimension size (number of outputs from embedding model)
	// Redis must know this value before creating vector index
	dimension := config.GetConfig().OllamaConfig.Dimension

	// 1. Configure and create Embedder
	// Acts as a translator,
	// specifically converting text to AI understandable vector representation
	embedConfig := &embeddingOllama.EmbeddingConfig{
		BaseURL: config.GetConfig().OllamaConfig.BaseURL, // Embedding model service address
		Model:   embeddingModel,                          // Which embedding model to use
	}

	// Create embedder instance
	// All subsequent text vectorization will be done through it
	embedder, err := embeddingOllama.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// ===============================
	// 2. Initialize vector index structure in Redis
	// ===============================
	// Prepare storage in Redis,
	// specifying that vectors of a certain dimension will be stored
	if err := redisPkg.InitRedisIndex(ctx, filename, dimension); err != nil {
		return nil, fmt.Errorf("failed to init redis index: %w", err)
	}

	// Get Redis client for subsequent data writing
	rdb := redisPkg.Rdb

	// ===============================
	// 3. Configure indexer (define how documents are stored in Redis)
	// ===============================
	indexerConfig := &redisIndexer.IndexerConfig{
		Client:    rdb,                                     // Redis client
		KeyPrefix: redis.GenerateIndexNamePrefix(filename), // Different knowledge bases use different prefixes to avoid conflicts
		BatchSize: 10,                                      // Batch process documents to improve write efficiency

		// Define how a Document is stored in Redis
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redisIndexer.Hashes, error) {

			// Extract source info (e.g. filename, URL) from document metadata
			source := ""
			if s, ok := doc.MetaData["source"].(string); ok {
				source = s
			}

			// Construct actual data structure (Hash) stored in Redis
			return &redisIndexer.Hashes{
				// Redis Key, usually composed of knowledge base name + document chunk ID
				Key: fmt.Sprintf("%s:%s", filename, doc.ID),

				// Fields in Redis Hash
				Field2Value: map[string]redisIndexer.FieldValue{
					// content: original text content
					// EmbedKey indicates this field needs vectorization first,
					// the generated vector will be stored in "vector" field
					"content": {Value: doc.Content, EmbedKey: "vector"},

					// metadata: auxiliary info, not participating in vector calculation
					"metadata": {Value: source},
				},
			}, nil
		},
	}

	// Pass embedder to indexer
	// This allows indexer to automatically vectorize when writing text
	indexerConfig.Embedding = embedder

	// ===============================
	// 4. Create final usable indexer instance
	// ===============================
	// Now indexer has capabilities for:
	// - Text to Vector
	// - Writing vectors to Redis
	idx, err := redisIndexer.NewIndexer(ctx, indexerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	// Return a wrapped RAGIndexer,
	// simply call it later to add documents to knowledge base
	return &RAGIndexer{
		embedding: embedder,
		indexer:   idx,
	}, nil
}

// IndexFile reads file content and creates vector index
func (r *RAGIndexer) IndexFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert file content to document
	// TODO: Can chunk text as needed, currently processing as a single document
	doc := &schema.Document{
		ID:      "doc_1", // Can use UUID or other unique identifier
		Content: string(content),
		MetaData: map[string]any{
			"source": filePath,
		},
	}

	// Use indexer to store document (will auto vectorize)
	_, err = r.indexer.Store(ctx, []*schema.Document{doc})
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

// DeleteIndex deletes knowledge base index for specified file (static method)
func DeleteIndex(ctx context.Context, filename string) error {
	if err := redisPkg.DeleteRedisIndex(ctx, filename); err != nil {
		return fmt.Errorf("failed to delete redis index: %w", err)
	}
	return nil
}

// NewRAGQuery creates RAG query instance (for vector retrieval and Q&A)
func NewRAGQuery(ctx context.Context, username string) (*RAGQuery, error) {
	cfg := config.GetConfig()

	// Create embedding model
	embedConfig := &embeddingOllama.EmbeddingConfig{
		BaseURL: cfg.OllamaConfig.BaseURL,
		Model:   cfg.OllamaConfig.EmbeddingModel,
	}
	embedder, err := embeddingOllama.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Get user uploaded filename (assuming one file per user)
	// Need to read filename from user directory here
	userDir := fmt.Sprintf("uploads/%s", username)
	files, err := os.ReadDir(userDir)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no uploaded file found for user %s", username)
	}

	var filename string
	for _, f := range files {
		if !f.IsDir() {
			filename = f.Name()
			break
		}
	}

	if filename == "" {
		return nil, fmt.Errorf("no valid file found for user %s", username)
	}

	// Create retriever
	rdb := redisPkg.Rdb
	indexName := redis.GenerateIndexName(filename)

	retrieverConfig := &redisRetriever.RetrieverConfig{
		Client:       rdb,
		Index:        indexName,
		Dialect:      2,
		ReturnFields: []string{"content", "metadata", "distance"},
		TopK:         5,
		VectorField:  "vector",
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == "content" {
					resp.Content = val
				} else {
					resp.MetaData[field] = val
				}
			}
			return resp, nil
		},
	}
	retrieverConfig.Embedding = embedder

	rtr, err := redisRetriever.NewRetriever(ctx, retrieverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	return &RAGQuery{
		embedding: embedder,
		retriever: rtr,
	}, nil
}

// RetrieveDocuments retrieves relevant documents
func (r *RAGQuery) RetrieveDocuments(ctx context.Context, query string) ([]*schema.Document, error) {
	docs, err := r.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}
	return docs, nil
}

// BuildRAGPrompt builds prompt containing retrieved documents
func BuildRAGPrompt(query string, docs []*schema.Document) string {
	if len(docs) == 0 {
		return query
	}

	contextText := ""
	for i, doc := range docs {
		contextText += fmt.Sprintf("[Document %d]: %s\n\n", i+1, doc.Content)
	}

	prompt := fmt.Sprintf(`Answer the user's question based on the following reference documents. If no relevant info is found, please state so.

Reference documents:
%s

User question: %s

Please provide an accurate and complete answer:`, contextText, query)

	return prompt
}
