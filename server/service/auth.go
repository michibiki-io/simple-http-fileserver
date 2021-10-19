package service

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/michibiki-io/simple-http-fileserver/server/model"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
	"github.com/tidwall/gjson"
)

var apiRetryCount int = 3

var apiMinTimeoutMillsecond int = 500

var apiMaxTimeoutMillsecond int = 1000

var ApiAccessTokenKey = "access_token"

var ApiRefreshTokenKey = "refresh_token"

var ApiExpireInKey = "expire_in"

func init() {

	apiRetryCount = utility.GetIntEnv("AUTH_SERVER_CALL_RETRY_COUNT", apiRetryCount)
	apiMinTimeoutMillsecond = utility.GetIntEnv("AUTH_SERVER_CALL_MIN_TIMEOUT", apiMinTimeoutMillsecond)
	apiMaxTimeoutMillsecond = utility.GetIntEnv("AUTH_SERVER_CALL_MAX_TIMEOUT", apiMaxTimeoutMillsecond)
	ApiAccessTokenKey = utility.
		GetEnv("AUTH_SERVER_ACCESS_TOKEN_JSON_PATH", ApiAccessTokenKey)
	ApiRefreshTokenKey = utility.
		GetEnv("AUTH_SERVER_REFRESH_TOKEN_JSON_PATH", ApiRefreshTokenKey)
	ApiExpireInKey = utility.
		GetEnv("AUTH_SERVER_EXPIREIN_JSON_PATH", ApiExpireInKey)

}

type ApiClientService struct{}

var client = resty.New()

func check(a *ApiClientService, c *resty.Client) (error error) {

	error = nil

	if a == nil {
		error = utility.NewError("instance is nil", utility.InternalServerError)
		return
	} else if client == nil {
		error = utility.NewError("resty client is nil", utility.InternalServerError)
		return
	}
	return

}

func callApiWrapper(uri string, param interface{}) (result map[string]interface{}, error error) {

	resp, err := client.
		SetRetryCount(apiRetryCount-1).
		SetRetryWaitTime(time.Duration(apiMinTimeoutMillsecond)*time.Millisecond).
		SetRetryMaxWaitTime(time.Duration(apiMaxTimeoutMillsecond)*time.Second).
		R().
		SetHeader("Content-Type", "application/json").
		SetBody(param).
		SetResult(map[string]interface{}{}).
		Post(uri)

	if err != nil {
		utility.Log.Debug("resty error occured: %s", err.Error())
		error = utility.NewError("no responce from auth server", utility.InternalServerError)
		return
	} else if resp.StatusCode() == http.StatusOK {
		result = (*resp.Result().(*map[string]interface{}))
		return
	} else if resp.StatusCode() == http.StatusUnauthorized {
		error = utility.NewError("no auth", utility.Unauthorized)
		return
	} else {
		utility.Log.Debug("resty error occured: %s", err.Error())
		error = utility.NewError("nother error", utility.InternalServerError)
		return
	}
}

func (a *ApiClientService) Authorize(auth model.Auth) (token model.TokenSet, error error) {

	token = model.TokenSet{}
	error = nil

	if err := check(a, client); err != nil {
		error = err
		return
	} else if result, err := callApiWrapper(utility.
		GetEnv("AUTH_SERVER_AUTH_URL", "http://localhost:80/v1/authorize"), auth); err != nil {
		error = err
		return
	} else {
		error = nil
		if access_token, ok := result[ApiAccessTokenKey]; ok {
			token.AccessToken = access_token.(string)
		}
		if refresh_token, ok := result[ApiRefreshTokenKey]; ok {
			token.RefreshToken = refresh_token.(string)
		}
		if expire_in, ok := result[ApiExpireInKey]; ok {
			if expire_in2, ok := expire_in.(float64); ok {
				token.Expires = time.Now().Add(time.Duration(int(expire_in2)) * time.Second).Unix()
			}
		}
		return
	}

}

func (a *ApiClientService) Verify(accessToken string) (user model.User, error error) {

	user = model.User{}
	error = nil

	if err := check(a, client); err != nil {
		error = err
		return
	} else if len(accessToken) == 0 {
		error = utility.NewError("access_token is empty", utility.Unauthorized)
		return
	} else if result, err := callApiWrapper(utility.
		GetEnv("AUTH_SERVER_VERIFY_URL", "http://localhost:80/v1/verify"), map[string]string{ApiAccessTokenKey: accessToken}); err != nil {
		error = err
		return
	} else {
		// result to json
		if bytes, err := json.Marshal(result); err != nil {
			error = utility.NewError("Auth server return unexpected value, not a json", utility.InternalServerError)
			return
		} else {
			if expire_in, ok := result[ApiExpireInKey]; ok {
				if expire_in2, ok := expire_in.(float64); ok {
					if expire_in2 > float64(utility.GetIntEnv("SESSION_STORE_USER_MAX_AGE", 300)) {
						expire_in2 = float64(utility.GetIntEnv("SESSION_STORE_USER_MAX_AGE", 300))
					}
					user.Expires = time.Now().Add(time.Duration(int(expire_in2)) * time.Second).Unix()
				}
			}
			// get user and groups
			if gjUser := gjson.Get(string(bytes), utility.
				GetEnv("AUTH_SERVER_USERID_JSON_PATH", "user.Id")); !gjUser.Exists() {
				error = utility.NewError("cannot get user auth info", utility.InternalServerError)
				user.Expires = 0
				return
			} else if gjGroup := gjson.Get(string(bytes), utility.
				GetEnv("AUTH_SERVER_GROUP_JSON_PATH", "user.Groups")); !gjGroup.Exists() {
				error = utility.NewError("cannot get user belonging group info", utility.InternalServerError)
				user.Expires = 0
				return
			} else {
				user.Id = gjUser.String()
				if err := json.Unmarshal([]byte(gjGroup.String()), &user.Groups); err != nil {
					utility.Log.Debug("group json is not a list")
				}
				return
			}
		}
	}
}

func (a *ApiClientService) Refresh(refreshToken string) (tokenSet model.TokenSet, error error) {

	tokenSet = model.TokenSet{}
	error = nil

	if err := check(a, client); err != nil {
		error = err
		return
	} else if len(refreshToken) == 0 {
		error = utility.NewError("refresh_token is empty", utility.Unauthorized)
		return
	} else if result, err := callApiWrapper(utility.
		GetEnv("AUTH_SERVER_REFRESH_URL", "http://localhost:80/v1/refresh"), map[string]string{ApiRefreshTokenKey: refreshToken}); err != nil {
		error = err
		return
	} else {
		// result to json
		if bytes, err := json.Marshal(result); err != nil {
			error = utility.NewError("Auth server return unexpected value, not a json", utility.InternalServerError)
			return
		} else {
			if expire_in, ok := result[ApiExpireInKey]; ok {
				if expire_in2, ok := expire_in.(float64); ok {
					tokenSet.Expires = time.Now().Add(time.Duration(int(expire_in2)) * time.Second).Unix()
				}
			}
			if gjAccessToken := gjson.Get(string(bytes), ApiAccessTokenKey); !gjAccessToken.Exists() {
				error = utility.NewError("cannot get access_token", utility.InternalServerError)
				tokenSet.Expires = 0
				return
			} else if gjRefreshToken := gjson.Get(string(bytes), ApiRefreshTokenKey); !gjRefreshToken.Exists() {
				error = utility.NewError("cannot get refresh_token", utility.InternalServerError)
				return
			} else {
				tokenSet.AccessToken = gjAccessToken.String()
				tokenSet.RefreshToken = gjRefreshToken.String()
				return
			}
		}
	}
}

func (a *ApiClientService) Deauthorize(accessToken string) (error error) {

	error = nil

	if err := check(a, client); err != nil {
		error = err
		return
	} else if len(accessToken) == 0 {
		error = utility.NewError("access_token is empty", utility.Unauthorized)
		return
	} else if _, err := callApiWrapper(utility.
		GetEnv("AUTH_SERVER_DEAUTH_URL", "http://localhost:80/v1/deauthorize"), map[string]string{ApiRefreshTokenKey: accessToken}); err != nil {
		error = err
		return
	} else {
		return
	}
}
