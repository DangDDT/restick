package user

import (
	"errors"
	"time"
)

const (
	RoleContributor = "contributor"
	RoleAdmin       = "admin"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string
	CreatedAt    time.Time
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

var (
	ErrInvalidRole = errors.New("invalid role")
	ErrEmailTaken  = errors.New("email taken")
)
