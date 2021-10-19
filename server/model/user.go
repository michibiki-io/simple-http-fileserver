package model

type User struct {
	Id      string
	Groups  []string
	Expires int64
}
