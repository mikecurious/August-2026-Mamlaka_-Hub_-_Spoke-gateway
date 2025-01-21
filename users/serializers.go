package users

import (
	"github.com/gin-gonic/gin"
)

type UserSerializer struct {
	c *gin.Context
	UserModel
}

type UsersSerializer struct {
	C     *gin.Context
	Users []UserModel
}

type UserResponse struct {
	UserModel
}

func NewUserSerializer(c *gin.Context, User UserModel) UserSerializer {
	return UserSerializer{
		c:         c,
		UserModel: User,
	}
}

func NewUsersSerializer(c *gin.Context, Users []UserModel) UsersSerializer {
	return UsersSerializer{
		C:     c,
		Users: Users,
	}
}

func (s *UserSerializer) Response() UserResponse {
	return UserResponse{
		UserModel: s.UserModel,
	}
}

func (s *UsersSerializer) Response() []UserResponse {
	response := make([]UserResponse, len(s.Users))
	for i, UserModel := range s.Users {
		serializer := NewUserSerializer(s.C, UserModel)
		response[i] = serializer.Response()
	}
	return response
}
