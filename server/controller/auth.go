package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michibiki-io/simple-http-fileserver/server/model"
	"github.com/michibiki-io/simple-http-fileserver/server/service"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
)

var folderPermissions map[string][]string = map[string][]string{}

var defaultPermission bool = false

func init() {

	// load private.json
	if bytes, err := ioutil.
		ReadFile(utility.GetEnv("FILE_PERMISSION_CONFIG_PATH", "config/permissions.json")); err != nil {
		utility.Log.Debug("folder permission config file is not found.")
	} else if err = json.Unmarshal(bytes, &folderPermissions); err != nil {
		utility.Log.Debug("cannot load folder permission config file.")
	}

	// default permission
	defaultPermission = utility.GetBoolEnv("FILE_PERMISSION_DEFAULT", defaultPermission)

}

func Authorize(c *gin.Context) {

	authModel := model.Auth{}

	if err := c.Bind(&authModel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		c.Abort()
		return
	}

	apiService := service.ApiClientService{}

	if tokenSet, err := apiService.Authorize(authModel); err != nil {
		c.JSON(errorToHttpStatus(err))
		c.Abort()
	} else {

		// response
		responce := gin.H{}

		if authModel.RedirectTo != "" {
			responce = gin.H{"redirect_to": authModel.RedirectTo}
		}

		// set
		c.Set("tokens", tokenSet)
		c.Set("response", responce)
	}

}

func AuthorizeToken(c *gin.Context) {

	mapToken := map[string]string{}
	if err := c.ShouldBindJSON(&mapToken); err != nil {
		c.JSON(http.StatusUnprocessableEntity, err.Error())
		return
	}

	tokenSet := model.TokenSet{}

	apiService := service.ApiClientService{}
	backendStore := service.BackendStoreService{}

	if apitoken, ok := mapToken["token"]; !ok {
		c.JSON(errorToHttpStatus(utility.NewError("token is required", utility.Forbidden)))
		c.Abort()
	} else if refreshToken, err := backendStore.Get(apitoken); err != nil {
		c.JSON(errorToHttpStatus(utility.NewError("token is invalid", utility.Unauthorized)))
		c.Abort()
	} else if tokenSet, err = apiService.Refresh(refreshToken); err != nil {
		c.JSON(errorToHttpStatus(utility.NewError("token is invalid", utility.Unauthorized)))
		c.Abort()
	} else {
		// delete used apitoken from backend
		backendStore.Del(apitoken)

		// set
		c.Set("tokens", tokenSet)
		c.Set("response", gin.H{
			service.ApiAccessTokenKey:  tokenSet.AccessToken,
			service.ApiRefreshTokenKey: tokenSet.RefreshToken,
			service.ApiExpireInKey:     tokenSet.Expires,
		})
	}

}

func Refresh(c *gin.Context) {

	tokenSet, ok := getTokenSetFromStore(c)

	if ok && tokenSet.RefreshToken != "" {

		apiService := service.ApiClientService{}

		if _tokenSet, err := apiService.Refresh(tokenSet.RefreshToken); err != nil {
			c.JSON(errorToHttpStatus(err))
			c.Abort()
		} else {
			c.Set("token", _tokenSet)
			if userModel, err := apiService.Verify(_tokenSet.AccessToken); err == nil {
				c.Set("user", userModel)
			}
			c.Set("response", gin.H{
				service.ApiAccessTokenKey:  _tokenSet.AccessToken,
				service.ApiRefreshTokenKey: _tokenSet.RefreshToken,
				service.ApiExpireInKey:     _tokenSet.Expires,
			})
			c.Next()
		}
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh_token is required."})
		c.Abort()
	}
}

func Deauthorize(c *gin.Context) {

	tokenSet, ok := getTokenSetFromStore(c)

	if ok && tokenSet.AccessToken != "" {

		apiService := service.ApiClientService{}

		if err := apiService.Deauthorize(tokenSet.AccessToken); err != nil {
			statusCode, message := errorToHttpStatus(err)
			if statusCode == http.StatusUnauthorized {
				c.Next()
			} else {
				c.JSON(statusCode, gin.H{"error": message})
				c.Abort()
			}
		} else {
			c.Next()
		}
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access_token is required."})
		c.Abort()
	}
}

func ProcessAccessToken(c *gin.Context) {

	// get from previous token
	tokenSet, _ := getTokenSetFromStore(c)

	authHeader := c.Request.Header.Get("authorization")
	collectValue := false
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenSet.AccessToken = strings.TrimPrefix(authHeader, "Bearer ")
		collectValue = true
	}

	mapToken := map[string]interface{}{}
	if err := c.ShouldBindJSON(&mapToken); err == nil {
		if access_token, ok := mapToken[service.ApiAccessTokenKey].(string); ok {
			tokenSet.AccessToken = access_token
			collectValue = true
		}
	}

	// set
	if collectValue {
		c.Set("tokens", tokenSet)
	}
}

func ProcessRefreshToken(c *gin.Context) {

	// get from previous token
	tokenSet, _ := getTokenSetFromStore(c)

	collectValue := false
	mapToken := map[string]interface{}{}
	if err := c.ShouldBindJSON(&mapToken); err == nil {
		if refresh_token, ok := mapToken[service.ApiRefreshTokenKey].(string); ok {
			tokenSet.RefreshToken = refresh_token
			collectValue = true
		}
	}

	// set
	if collectValue {
		c.Set("tokens", tokenSet)
	}
}

func StatusCheck(c *gin.Context) {

	// get user info from cookie
	_, userFound := getUserFromStore(c)

	if userFound {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "ng"})
	}
}

func VerifyAuth(c *gin.Context) {

	// get user info from cookie
	userModel, userFound := getUserFromStore(c)

	if !userFound || userModel.Expires == 0 || userModel.Id == "" {

		tokenSet, ok := getTokenSetFromStore(c)

		// responce json
		responce := gin.H{}

		// token verify and refresh
		var verifyError error = nil
		var refreshError error = nil

		if ok && tokenSet.AccessToken != "" {

			// api service
			apiService := service.ApiClientService{}

			// access_token
			userModel, verifyError = apiService.Verify(tokenSet.AccessToken)

			// use refreshToken
			if verifyError != nil {
				if _tokenSet, err := apiService.Refresh(tokenSet.RefreshToken); err != nil {
					responce = gin.H{"error": err.Error()}
					refreshError = err
				} else {
					c.Set("tokens", _tokenSet)
					if userModel, verifyError = apiService.Verify(_tokenSet.AccessToken); verifyError == nil {
						c.Set("user", userModel)
					}
				}
			} else {
				c.Set("user", userModel)
			}
		}

		if tokenSet.AccessToken == "" || verifyError != nil || refreshError != nil {
			if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
				c.Next()
				return
			} else {
				c.JSON(http.StatusUnauthorized, responce)
				c.Abort()
				return
			}
		}
	} else {
		c.Set("user", userModel)
	}
}

func ShowAuthorizeInterfaceHander(url string) gin.HandlerFunc {

	return func(c *gin.Context) {

		// get user info from cookie
		userModel := model.User{}

		// get user model from previous func
		userFound := false
		if tmp, ok := c.Get("user"); ok {
			userModel, _ = tmp.(model.User)
			userFound = ok
		}

		if !userFound || userModel.Expires == 0 || userModel.Id == "" {
			c.HTML(http.StatusOK, "login", gin.H{
				"request_api": url,
				"redirect_to": c.Request.URL.Path,
				"contextPath": utility.GetContextPath()})
			c.Abort()
		} else {
			c.Next()
		}
	}
}

func isPermitted(url string) (groups []string) {

	groups = []string{}

	groups, ok := folderPermissions[url]
	if !ok {
		for key, tmp := range folderPermissions {
			if strings.HasSuffix(key, "*") {
				prefix := strings.TrimSuffix(key, "*")
				if url == prefix || strings.HasPrefix(url, prefix) {
					groups = tmp
					return
				}
			}
		}
	}

	return
}

func errorToHttpStatus(error error) (statusCode int, message gin.H) {
	if error, ok := error.(*utility.Error); ok {
		errNo := error.No()

		statusCode = http.StatusUnauthorized
		message = gin.H{"error": error.Error(), "no": errNo}

		switch errNo {
		case utility.Unauthorized:
			statusCode = http.StatusUnauthorized
		case utility.Forbidden:
			statusCode = http.StatusForbidden
		default:
			statusCode = http.StatusUnauthorized
		}
	} else {
		statusCode = http.StatusInternalServerError
		message = gin.H{"error": error.Error()}
	}

	return
}

func getTokenSetFromStore(c *gin.Context) (tokenSet model.TokenSet, exists bool) {

	tokenSet = model.TokenSet{}
	exists = false

	if tmp, ok := c.Get("tokens"); ok {
		if tokenSet, ok = tmp.(model.TokenSet); !ok {
			if jsonString, ok := tmp.(string); ok {
				if err := json.Unmarshal([]byte(jsonString), &tokenSet); err == nil {
					exists = true
				}
			}
		} else {
			exists = true
		}
	}

	return
}

func getUserFromStore(c *gin.Context) (userModel model.User, exists bool) {

	userModel = model.User{}
	exists = false

	if tmp, ok := c.Get("user"); ok {
		if userModel, ok = tmp.(model.User); !ok {
			if jsonString, ok := tmp.(string); ok {
				json.Unmarshal([]byte(jsonString), &userModel)
			}
		}
		if userModel.Expires > time.Now().Unix() {
			exists = true
		} else {
			userModel = model.User{}
			exists = false
		}
	}

	return
}
