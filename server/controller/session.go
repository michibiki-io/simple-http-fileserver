package controller

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/michibiki-io/simple-http-fileserver/server/service"
)

func CreateSessionHandler() (gin.HandlerFunc, error) {

	sessionService := service.SessionService{}

	if store, err := sessionService.GetSessionStore(); err != nil {
		return nil, err
	} else {
		return sessions.Sessions("session", *store), nil
	}
}
