package service

import (
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	sessionRedis "github.com/gin-contrib/sessions/redis"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
)

var store *sessions.Store = nil

type SessionService struct{}

func init() {
	sessionBackend := strings.ToLower(utility.GetEnv("SESSION_BACKEND", "cookie"))

	var backendInitCompleted bool = false

	// redis based
	if sessionBackend == "redis" {
		if redisStore, err := sessionRedis.NewStore(10, "tcp",
			utility.GetEnv("SESSION_BACKEND_HOST", "localhost:6379"),
			utility.GetEnv("SESSION_BACKEND_PASSWORD", ""),
			[]byte(utility.RandomString(32))); err == nil {
			if sessionStore, ok := redisStore.(sessions.Store); ok {
				store = &sessionStore
				backendInitCompleted = true
			}
		} else {
			utility.Log.Debug("Session backend [%s] initialize faild, %s", sessionBackend, err)
		}
	}

	if !backendInitCompleted {
		cookieStore := cookie.NewStore([]byte(utility.RandomString(64)))
		cookieStore.Options(sessions.Options{
			Domain:   "*",
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 7,
			HttpOnly: true,
		})
		if sessionStore, ok := cookieStore.(sessions.Store); ok {
			store = &sessionStore
		}
	}
}

func (s *SessionService) GetSessionStore() (*sessions.Store, error) {

	if s == nil || store == nil {
		return nil, utility.NewError("instance is null", utility.InternalServerError)
	}

	return store, nil

}
