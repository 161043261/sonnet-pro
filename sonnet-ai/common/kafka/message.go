package kafka

import (
	"encoding/json"
	"lark_ai/dao/message"
	"lark_ai/model"
)

type MessageMQParam struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func GenerateMessageMQParam(sessionID string, content string, username string, IsUser bool) []byte {
	param := MessageMQParam{
		SessionID: sessionID,
		Content:   content,
		UserName:  username,
		IsUser:    IsUser,
	}
	data, _ := json.Marshal(param)
	return data
}

func MQMessage(msg []byte) error {
	var param MessageMQParam
	err := json.Unmarshal(msg, &param)
	if err != nil {
		return err
	}
	newMsg := &model.Message{
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		IsUser:    param.IsUser,
	}
	// Consumer asynchronously inserts into database
	message.CreateMessage(newMsg)
	return nil
}
