package router

import (
	"fmt"
	"lark_ai/middleware/jwt"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func InitRouter(addr string, port int) *server.Hertz {

	r := server.Default(server.WithHostPorts(fmt.Sprintf("%s:%d", addr, port)))

	enterRouter := r.Group("/api/v1")
	{
		RegisterUserRouter(enterRouter.Group("/user"))
	}
	// Subsequent login APIs require JWT authentication
	{
		AIGroup := enterRouter.Group("/ai")
		AIGroup.Use(jwt.Auth())
		AIRouter(AIGroup)
	}

	{
		FileGroup := enterRouter.Group("/file")
		FileGroup.Use(jwt.Auth())
		FileRouter(FileGroup)
	}

	return r
}
