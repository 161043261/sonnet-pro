package user

import (
	"context"
	"lark_ai/common/mysql"
	"lark_ai/model"
	"lark_ai/utils"

	"gorm.io/gorm"
)

const (
	CodeMsg     = "lark_ai verification code (valid for 2 minutes): "
	UserNameMsg = "lark_ai account info, please keep it safe for future login: "
)

var ctx = context.Background()

// Login via account only
func IsExistUser(username string) (bool, *model.User) {

	user, err := mysql.GetUserByUsername(username)

	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}

	return true, user
}

func Register(username, email, password string) (*model.User, bool) {
	if user, err := mysql.InsertUser(&model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: utils.MD5(password),
	}); err != nil {
		return nil, false
	} else {
		return user, true
	}
}
