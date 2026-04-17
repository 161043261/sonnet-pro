package file

import (
	"context"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"lark_ai/common/rag"
	"lark_ai/config"
	"lark_ai/utils"
)

// Upload RAG related files (only text files allowed)
// Vectorizing and saving directly is possible, but keeping it on server allows viewing historical RAG files
func UploadRagFile(username string, file *multipart.FileHeader) (string, error) {
	// Validate file type and filename
	if err := utils.ValidateFile(file); err != nil {
		log.Printf("File validation failed: %v", err)
		return "", err
	}

	// Create user directory
	userDir := filepath.Join("uploads", username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Failed to create user directory %s: %v", userDir, err)
		return "", err
	}

	// Delete all existing files and indexes in user directory (one file per user)
	files, err := os.ReadDir(userDir)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() {
				filename := f.Name()
				// Delete Redis index corresponding to the file
				if err := rag.DeleteIndex(context.Background(), filename); err != nil {
					log.Printf("Failed to delete index for %s: %v", filename, err)
					// Continue execution, do not interrupt upload due to index deletion failure
				}
			}
		}
	}
	// Delete all files in user directory
	if err := utils.RemoveAllFilesInDir(userDir); err != nil {
		log.Printf("Failed to clean user directory %s: %v", userDir, err)
		return "", err
	}

	// Generate UUID as unique filename
	uuid := utils.GenerateUUID()

	ext := filepath.Ext(file.Filename)
	filename := uuid + ext
	filePath := filepath.Join(userDir, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		log.Printf("Failed to open uploaded file: %v", err)
		return "", err
	}
	defer src.Close()

	// Create target file
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create destination file %s: %v", filePath, err)
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		log.Printf("Failed to copy file content: %v", err)
		return "", err
	}

	log.Printf("File uploaded successfully: %s", filePath)

	// Create RAG indexer and vectorize file
	indexer, err := rag.NewRAGIndexer(filename, config.GetConfig().OllamaConfig.EmbeddingModel)
	if err != nil {
		log.Printf("Failed to create RAG indexer: %v", err)
		// Delete uploaded file
		os.Remove(filePath)
		return "", err
	}

	// Read file content and create vector index
	if err := indexer.IndexFile(context.Background(), filePath); err != nil {
		log.Printf("Failed to index file: %v", err)
		// Delete uploaded file and index
		os.Remove(filePath)
		rag.DeleteIndex(context.Background(), filename)
		return "", err
	}

	log.Printf("File indexed successfully: %s", filename)
	return filePath, nil
}
