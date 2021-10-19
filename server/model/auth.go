package model

type Auth struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RedirectTo string `json:"redirect_to"`
}

type AuthorizationState struct {
	StatusCode int
}
