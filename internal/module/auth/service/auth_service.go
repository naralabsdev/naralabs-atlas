package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/repository"
	mailemail "github.com/naralabs/naralabs-atlas/lib/email"
)

type AuthService struct {
	cfg    *config.Config
	repo   *repository.AuthRepository
	mailer mailemail.Sender
	now    func() time.Time
}

func NewAuthService(cfg *config.Config, repo *repository.AuthRepository, mailer mailemail.Sender) *AuthService {
	return &AuthService{
		cfg:    cfg,
		repo:   repo,
		mailer: mailer,
		now:    time.Now,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, callbackURL string) (model.RegisterResult, error) {
	if err := validateEmail(email); err != nil {
		return model.RegisterResult{}, err
	}
	if err := validatePassword(password); err != nil {
		return model.RegisterResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.RegisterResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.RegisterResult{}, ErrEmailExists
		}
		return model.RegisterResult{}, err
	}

	if err := s.sendVerificationEmail(ctx, user, callbackURL); err != nil {
		return model.RegisterResult{}, err
	}

	return model.RegisterResult{
		User:                 user.Public(),
		RequiresVerification: true,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (model.AuthSession, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return model.AuthSession{}, ErrInvalidCredentials
	}
	if err != nil {
		return model.AuthSession{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return model.AuthSession{}, ErrInvalidCredentials
	}
	if user.EmailVerifiedAt == nil {
		return model.AuthSession{}, ErrEmailNotVerified
	}

	return s.issueSession(user)
}

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) (model.AuthSession, error) {
	tokenHash, err := hashToken(rawToken)
	if err != nil {
		return model.AuthSession{}, ErrInvalidToken
	}

	user, err := s.repo.ConsumeVerificationToken(ctx, tokenHash, s.now())
	if errors.Is(err, repository.ErrUserNotFound) {
		return model.AuthSession{}, ErrInvalidToken
	}
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "expired") {
		return model.AuthSession{}, ErrTokenExpired
	}
	if err != nil {
		return model.AuthSession{}, err
	}

	return s.issueSession(user)
}

func (s *AuthService) ResendVerification(ctx context.Context, email, callbackURL string) (bool, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if user.EmailVerifiedAt != nil {
		return true, nil
	}

	latest, err := s.repo.LatestActiveVerificationCreatedAt(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if latest != nil {
		elapsed := s.now().Sub(*latest)
		if elapsed < s.cfg.Auth.VerificationResendCooldown {
			return false, ErrResendCooldown
		}
	}

	if err := s.sendVerificationEmail(ctx, user, callbackURL); err != nil {
		return false, err
	}
	return false, nil
}

func (s *AuthService) Me(ctx context.Context, token string) (model.UserPublic, error) {
	userID, err := s.parseToken(token)
	if err != nil {
		return model.UserPublic{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return model.UserPublic{}, ErrUnauthorized
	}
	if err != nil {
		return model.UserPublic{}, err
	}
	return user.Public(), nil
}

func (s *AuthService) sendVerificationEmail(ctx context.Context, user model.User, callbackURL string) error {
	rawToken, tokenHash, expiresAt, err := s.newVerificationToken()
	if err != nil {
		return err
	}

	now := s.now()
	if err := s.repo.InvalidateActiveVerificationTokens(ctx, user.ID, now); err != nil {
		return err
	}
	if err := s.repo.CreateVerificationToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return err
	}

	verifyURL, err := s.buildVerifyURL(rawToken, callbackURL)
	if err != nil {
		return err
	}

	return s.mailer.SendVerificationEmail(ctx, mailemail.VerificationMailInput{
		To:          user.Email,
		VerifyURL:   verifyURL,
		ExpiresIn:   s.cfg.Auth.VerificationTokenTTL,
		ProductName: s.cfg.ServiceName,
	})
}

func (s *AuthService) buildVerifyURL(rawToken, callbackURL string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.Auth.WebAppURL), "/")
	if base == "" {
		base = "http://localhost:3000"
	}

	u, err := url.Parse(base + "/api/auth/verify-email")
	if err != nil {
		return "", fmt.Errorf("parse verify url: %w", err)
	}

	q := u.Query()
	q.Set("token", rawToken)
	if safe := sanitizeCallbackURL(callbackURL, base); safe != "" {
		q.Set("callbackUrl", safe)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *AuthService) issueSession(user model.User) (model.AuthSession, error) {
	expiresAt := s.now().Add(s.cfg.Auth.JWTExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   expiresAt.Unix(),
		"iat":   s.now().Unix(),
	})

	signed, err := token.SignedString([]byte(s.cfg.Auth.JWTSecret))
	if err != nil {
		return model.AuthSession{}, fmt.Errorf("sign jwt: %w", err)
	}

	return model.AuthSession{
		Token:     signed,
		ExpiresAt: expiresAt,
		User:      user.Public(),
	}, nil
}

func (s *AuthService) parseToken(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "Bearer ")
	if raw == "" {
		return "", ErrUnauthorized
	}

	parsed, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return "", ErrUnauthorized
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrUnauthorized
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", ErrUnauthorized
	}
	return sub, nil
}

func (s *AuthService) newVerificationToken() (raw string, hash string, expiresAt time.Time, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate token: %w", err)
	}
	raw = hex.EncodeToString(buf)
	hash, err = hashToken(raw)
	if err != nil {
		return "", "", time.Time{}, err
	}
	expiresAt = s.now().Add(s.cfg.Auth.VerificationTokenTTL)
	return raw, hash, expiresAt, nil
}

func hashToken(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty token")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	for _, r := range password {
		switch {
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit {
		return fmt.Errorf("password must contain at least one number, one uppercase, and one lowercase letter")
	}
	return nil
}

func sanitizeCallbackURL(raw, base string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return ""
	}
	return raw
}
