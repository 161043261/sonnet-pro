package router

import (
	"lark_ai/controller/file"

	"github.com/cloudwego/hertz/pkg/route"
)

func FileRouter(r *route.RouterGroup) {
	r.POST("/upload", file.UploadRagFile)
}
