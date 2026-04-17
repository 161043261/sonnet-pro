package user

import (
	"lark_ai/common/code"
	"lark_ai/controller"
	"lark_ai/service/user"

	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type (
	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	LoginResponse struct {
		controller.Response
		Token string `json:"token,omitempty"`
	}
	RegisterRequest struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password"`
	}
	RegisterResponse struct {
		controller.Response
		Token string `json:"token,omitempty"`
	}
)

func Login(ctx context.Context, c *app.RequestContext) {

	req := new(LoginRequest)
	res := new(LoginResponse)
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}

	token, code_ := user.Login(req.Username, req.Password)
	if code_ != code.CodeSuccess {
		c.JSON(200, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Token = token
	c.JSON(200, res)

}

func Register(ctx context.Context, c *app.RequestContext) {

	req := new(RegisterRequest)
	res := new(RegisterResponse)
	if err := c.BindAndValidate(req); err != nil {
		c.JSON(200, res.CodeOf(code.CodeInvalidParams))
		return
	}

	token, code_ := user.Register(req.Email, req.Password)
	if code_ != code.CodeSuccess {
		c.JSON(200, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Token = token
	c.JSON(200, res)
}
