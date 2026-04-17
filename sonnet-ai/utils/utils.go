package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"mime/multipart"
	"os"
	"path/filepath"
	"lark_ai/model"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

func GetRandomNumbers(num int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	code := ""
	for i := 0; i < num; i++ {
		// 0~9 random number
		digit := r.Intn(10)
		code += strconv.Itoa(digit)
	}
	return code
}

// MD5 encryption
func MD5(str string) string {
	m := md5.New()
	m.Write([]byte(str))
	return hex.EncodeToString(m.Sum(nil))
}

func GenerateUUID() string {
	return uuid.New().String()
}

// Convert schema message to database storable format
func ConvertToModelMessage(sessionID string, username string, msg *schema.Message) *model.Message {
	return &model.Message{
		SessionID: sessionID,
		UserName:  username,
		Content:   msg.Content,
	}
}

// Convert database message to schema message (for AI usage)
func ConvertToSchemaMessages(msgs []*model.Message) []*schema.Message {
	schemaMsgs := make([]*schema.Message, 0, len(msgs))
	for _, m := range msgs {
		role := schema.Assistant
		if m.IsUser {
			role = schema.User
		}
		schemaMsgs = append(schemaMsgs, &schema.Message{
			Role:    role,
			Content: m.Content,
		})
	}
	return schemaMsgs
}

// RemoveAllFilesInDir deletes all files in directory (excluding subdirectories)
func RemoveAllFilesInDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Ignore if directory does not exist
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(dir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateFile checks if file is an allowed text file (.md or .txt)
func ValidateFile(file *multipart.FileHeader) error {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".md" && ext != ".txt" {
		return fmt.Errorf("invalid file type, only .md or .txt allowed, current extension: %s", ext)
	}

	return nil
}
