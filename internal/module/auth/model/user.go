package model

import "time"

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UserPublic struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"emailVerified"`
	VerifiedAt    *time.Time `json:"verifiedAt,omitempty"`
}

func (u User) Public() UserPublic {
	return UserPublic{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerifiedAt != nil,
		VerifiedAt:    u.EmailVerifiedAt,
	}
}

type AuthSession struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	User      UserPublic `json:"user"`
}

type RegisterResult struct {
	User                 UserPublic `json:"user"`
	RequiresVerification bool       `json:"requiresVerification"`
}
