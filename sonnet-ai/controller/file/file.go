package file

import (
	"log"
	"lark_ai/common/code"
	"lark_ai/controller"
	"lark_ai/service/file"

	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type (
	UploadFileResponse struct {
		FilePath string `json:"file_path,omitempty"`
		controller.Response
	}
)

func UploadRagFile(ctx context.Context, c *app.RequestContext) {
	res := new(UploadFileResponse)
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		log.Println("FormFile fail ", err)
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}

	username := c.GetString("username")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(200, res.CodeOf(code.CodeInvalidToken))
		return
	}

	// indexer will be created in service layer based on actual filename
	filePath, err := file.UploadRagFile(username, uploadedFile)
	if err != nil {
		log.Println("UploadFile fail ", err)
		c.JSON(200, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.FilePath = filePath
	c.JSON(200, res)
}
