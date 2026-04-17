package jwt

import (
	"log"
	"lark_ai/common/code"
	"lark_ai/controller"
	lark_jwt "lark_ai/utils/jwt"
	"strings"

	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// Read JWT
func Auth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		res := new(controller.Response)

		var token string
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Support token via URL parameter
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(200, res.CodeOf(code.CodeInvalidToken))
			c.Abort()
			return
		}

		log.Println("token is ", token)
		username, ok := lark_jwt.ParseToken(token)
		if !ok {
			c.JSON(200, res.CodeOf(code.CodeInvalidToken))
			c.Abort()
			return
		}

		c.Set("username", username)
		c.Next(ctx)
	}
}
