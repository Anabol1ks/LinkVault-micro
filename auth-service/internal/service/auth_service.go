package service

import (
	"errors"
	"linkv-auth/config"
	"linkv-auth/internal/jwt"
	"linkv-auth/internal/models"
	"linkv-auth/internal/repository"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserExists = errors.New("user already exists")

type UserService struct {
	repo              *repository.UserRepository
	rtRepo            *repository.RefreshTokenRepository
	emailTokenRepo    *repository.EmailVerificationTokenRepository
	passwordResetRepo *repository.PasswordResetTokenRepository
	Cfg               *config.Config
}

func NewUserService(
	repo *repository.UserRepository,
	rtRepo *repository.RefreshTokenRepository,
	emailTokenRepo *repository.EmailVerificationTokenRepository,
	passwordResetRepo *repository.PasswordResetTokenRepository,
	cfg *config.Config,
) *UserService {
	return &UserService{
		repo:              repo,
		rtRepo:            rtRepo,
		emailTokenRepo:    emailTokenRepo,
		passwordResetRepo: passwordResetRepo,
		Cfg:               cfg,
	}
}

func (s *UserService) Register(name, email, password string) (*models.User, error) {
	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, ErrUserExists
	}

	hashedPassword, err := hashedPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashedPassword,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	emailVerToken := &models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.emailTokenRepo.Create(emailVerToken); err != nil {
		return nil, err
	}

	return user, nil
}

func hashedPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidPassword = errors.New("invalid password")

func (s *UserService) Login(email, password string) (access, refresh string, err error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return "", "", ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidPassword
	}

	access, accessClaims, err := jwt.GenerateAccessToken(user.ID.String(), &s.Cfg.JWT)
	if err != nil {
		return "", "", err
	}

	refresh, refreshClaims, err := jwt.GenerateRefreshToken(user.ID.String(), &s.Cfg.JWT)
	if err != nil {
		return "", "", err
	}

	if err := s.rtRepo.Create(&models.RefreshToken{
		JTI:       refreshClaims.ID,
		UserID:    user.ID,
		ExpiresAt: refreshClaims.ExpiresAt.Time,
		Revoked:   false,
	}); err != nil {
		return "", "", err
	}

	_ = accessClaims
	return access, refresh, nil
}

var ErrInvalidToken = errors.New("invalid token")

func (s *UserService) Refresh(refreshToken string) (access, refresh string, err error) {
	claims, err := jwt.ParseRefreshToken(refreshToken, s.Cfg.JWT.Refresh)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	rt, err := s.rtRepo.FindValid(claims.ID)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	_ = s.rtRepo.RevokeByJTI(rt.JTI) // игнорируем ошибку

	access, accessClaims, err := jwt.GenerateAccessToken(claims.UserID, &s.Cfg.JWT)
	if err != nil {
		return "", "", err
	}
	refresh, refreshClaims, err := jwt.GenerateRefreshToken(claims.UserID, &s.Cfg.JWT)
	if err != nil {
		return "", "", err
	}
	if err := s.rtRepo.Create(&models.RefreshToken{
		JTI:       refreshClaims.ID,
		UserID:    uuid.MustParse(claims.UserID),
		ExpiresAt: refreshClaims.ExpiresAt.Time,
		Revoked:   false,
	}); err != nil {
		return "", "", err
	}
	_ = accessClaims
	return access, refresh, nil
}

func (s *UserService) Profile(userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Logout(userID uuid.UUID) error {
	if err := s.rtRepo.RevokeAllForUser(userID); err != nil {
		return err
	}
	return nil
}

func (s *UserService) VerifyEmail(token string) error {
	emailToken, err := s.emailTokenRepo.FindByToken(token)
	if err != nil {
		return ErrInvalidToken
	}

	if err := s.repo.MarkEmailVerified(emailToken.UserID); err != nil {
		return err
	}

	if err := s.emailTokenRepo.MarkUsed(emailToken.ID); err != nil {
		return err
	}

	return nil
}

var ErrEmailAlready = errors.New("email already verified")

func (s *UserService) ResendVerificationEmail(userID uuid.UUID) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.EmailVerified {
		return ErrEmailAlready
	}

	emailVerToken := &models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.emailTokenRepo.Create(emailVerToken); err != nil {
		return err
	}
	return nil
}

func (s *UserService) RequestPasswordReset(email string) error {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return ErrUserNotFound
	}

	latest, err := s.passwordResetRepo.FindLatestByUser(user.ID)
	if err == nil && latest != nil && !latest.Used && latest.ExpiresAt.After(time.Now()) {
		return nil
	}

	passwordResetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.passwordResetRepo.Create(passwordResetToken); err != nil {
		return err
	}

	return nil
}

func (s *UserService) ConfirmPasswordReset(token, newPassword string) error {
	prt, err := s.passwordResetRepo.FindByToken(token)
	if err != nil {
		return ErrInvalidToken
	}
	user, err := s.repo.FindByID(prt.UserID)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := hashedPassword(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := s.repo.UpdatePassword(user); err != nil {
		return err
	}
	if err := s.passwordResetRepo.MarkUsed(prt.ID); err != nil {
		return err
	}
	return nil
}
