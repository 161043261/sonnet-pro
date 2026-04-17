package user

import (
	"lark_ai/common/code"
	"lark_ai/dao/user"
	"lark_ai/model"
	"lark_ai/utils"

	lark_jwt "lark_ai/utils/jwt"
)

func Login(username, password string) (string, code.Code) {
	var userInformation *model.User
	var ok bool
	if ok, userInformation = user.IsExistUser(username); !ok {

		return "", code.CodeUserNotExist
	}
	if userInformation.Password != utils.MD5(password) {
		return "", code.CodeInvalidPassword
	}
	token, err := lark_jwt.GenerateToken(userInformation.ID, userInformation.Username)

	if err != nil {
		return "", code.CodeServerBusy
	}
	return token, code.CodeSuccess
}

func Register(email, password string) (string, code.Code) {

	var ok bool
	var userInformation *model.User

	if ok, _ := user.IsExistUser(email); ok {
		return "", code.CodeUserExist
	}

	username := email

	if userInformation, ok = user.Register(username, email, password); !ok {
		return "", code.CodeServerBusy
	}

	token, err := lark_jwt.GenerateToken(userInformation.ID, userInformation.Username)

	if err != nil {
		return "", code.CodeServerBusy
	}

	return token, code.CodeSuccess
}
