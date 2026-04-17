package router

import (
	"lark_ai/controller/session"

	"github.com/cloudwego/hertz/pkg/route"
)

func AIRouter(r *route.RouterGroup) {
	{
		r.GET("/chat/sessions", session.GetUserSessionsByUserName)
		r.POST("/chat/send-new-session", session.CreateSessionAndSendMessage)
		r.POST("/chat/send", session.ChatSend)
		r.POST("/chat/history", session.ChatHistory)

		r.POST("/chat/send-stream-new-session", session.CreateStreamSessionAndSendMessage)
		r.POST("/chat/send-stream", session.ChatStreamSend)
	}

}
