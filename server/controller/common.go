package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func FromStoreToSessionHandler(fromStoreKey, toSessionKey string) gin.HandlerFunc {

	return func(c *gin.Context) {

		if tmp, ok := c.Get("isPublic"); ok {
			if isPublic, _ := tmp.(bool); isPublic {
				c.Next()
				return
			}
		}

		session := sessions.Default(c)

		if result, ok := c.Get(fromStoreKey); ok {
			if bytes, err := json.Marshal(result); err == nil {
				if unquoteString, err := strconv.Unquote(string(bytes)); err == nil {
					session.Set(toSessionKey, unquoteString)
				} else {
					session.Set(toSessionKey, string(bytes))
				}
				session.Save()
			}
		}
	}
}

func FromSessionToStoreHandler(fromSessionKey, toStoreKey string) gin.HandlerFunc {

	return func(c *gin.Context) {

		if tmp, ok := c.Get("isPublic"); ok {
			if isPublic, _ := tmp.(bool); isPublic {
				c.Next()
				return
			}
		}

		session := sessions.Default(c)

		if jsonString, ok := session.Get(fromSessionKey).(string); ok {
			c.Set(toStoreKey, jsonString)
		}
	}
}

func ClearSession(c *gin.Context) {

	session := sessions.Default(c)
	session.Clear()
	session.Save()

}

func JsonResponceHandler(storeKey string) gin.HandlerFunc {

	return func(c *gin.Context) {

		if result, ok := c.Get(storeKey); ok {
			c.JSON(http.StatusOK, result)
		}
	}
}
