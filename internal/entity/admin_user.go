package entity

import "time"

type AdminUser struct {
	ID           int64
	Email        string
	PasswordHash string // bcrypt; never serialized
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
