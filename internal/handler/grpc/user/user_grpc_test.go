package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRegisterUserApiServerRegistersService(t *testing.T) {
	server := grpc.NewServer()

	RegisterUserApiServer(server, &UnimplementedUserApiServer{})

	services := server.GetServiceInfo()
	info, ok := services["user.UserApi"]

	assert.True(t, ok)
	assert.Len(t, info.Methods, 10) // Register, Login, Auth0Login, RefreshToken, GetMe, Logout, ListUsers, GetUser, UpdateUser, DeleteUser
}
