package router

import (
	"lark_ai/controller/user"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterUserRouter(r *route.RouterGroup) {
	{
		r.POST("/register", user.Register)
		r.POST("/login", user.Login)
	}
}
