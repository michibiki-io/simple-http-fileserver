package controller

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/michibiki-io/simple-http-fileserver/server/model"
	"github.com/michibiki-io/simple-http-fileserver/server/service"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
)

func RequestApiToken(c *gin.Context) {

	user := model.User{}

	backendStore := service.BackendStoreService{}
	if !backendStore.IsServiceAvailable() {
		utility.Log.Debug("backend service is not available, api token system is not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "you cannnot access api token"})
		c.Abort()
	}

	// get tokenset from cookie
	tokenSet := model.TokenSet{}
	if tokenJson, ok := c.Get("tokens"); ok {
		if tokenSet, ok = tokenJson.(model.TokenSet); !ok {
			if tokenString, ok := tokenJson.(string); ok {
				json.Unmarshal([]byte(tokenString), &tokenSet)
			}
		}
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you cannnot access api token"})
		c.Abort()
		return
	}

	// refresh token
	apiService := service.ApiClientService{}
	tokenSet, err := apiService.Refresh(tokenSet.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you cannnot access api token"})
		c.Abort()
		return
	} else {
		c.Set("tokens", tokenSet)
	}

	// access token
	user, err = apiService.Verify(tokenSet.AccessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you cannnot access api token"})
		c.Abort()
		return
	}

	apitoken := shortuuid.New()
	expire_in := utility.GetIntEnv("API_TOKEN_EXPIRE_IN", 600)
	expire := time.Now().Add(time.Duration(expire_in) * time.Second).Unix()
	if err := backendStore.Set(apitoken, tokenSet.RefreshToken, time.Duration(expire_in)*time.Second); err != nil {
		utility.Log.Debug("backend service is not available, api token system is not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "you cannnot access api token"})
		c.Abort()
	}

	c.Set("response", gin.H{"user": user, "apitoken": apitoken, "expire": expire, "expire_in": expire_in})

}
