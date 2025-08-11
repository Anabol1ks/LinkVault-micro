package service

import (
	"errors"
	"link-service/internal/models"
	"link-service/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
)

type ShortLinkService struct {
	repo *repository.ShortLinkRepository
	Log  *zap.Logger
}

func NewShortLinkService(repo *repository.ShortLinkRepository, log *zap.Logger) *ShortLinkService {
	return &ShortLinkService{
		repo: repo,
		Log:  log,
	}
}

var ErrGenerateShortCode = errors.New("error generating short code")
var ErrCreateShortLink = errors.New("error creating short link")

func (s *ShortLinkService) CreateShortLink(originalURL string, userID *uuid.UUID, expireAfter *time.Duration) (*models.ShortLink, error) {
	var finalExpireAt *time.Time
	if expireAfter != nil {
		exp := time.Now().Add(*expireAfter)
		finalExpireAt = &exp
	} else if userID == nil {
		exp := time.Now().Add(7 * 24 * time.Hour)
		finalExpireAt = &exp
	} else {
		finalExpireAt = nil
	}

	shortCode, err := generateShortCode()
	if err != nil {
		s.Log.Error("Failed to generate short code", zap.Error(err))
		return nil, ErrGenerateShortCode
	}

	shortLink := &models.ShortLink{
		OriginalURL: originalURL,
		UserID:      userID,
		ShortCode:   shortCode,
		IsActive:    true,
		ExpireAt:    finalExpireAt,
	}

	if err := s.repo.Create(shortLink); err != nil {
		s.Log.Error("Failed to create short link", zap.Error(err))
		return nil, ErrCreateShortLink
	}

	return shortLink, nil
}

func generateShortCode() (string, error) {
	id, err := shortid.Generate()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *ShortLinkService) GetOriginalURL(shortCode string) (string, error) {
	url, err := s.repo.GetOriginalURL(shortCode)
	if err != nil {
		s.Log.Warn("Short link not found or inactive/expired", zap.String("shortCode", shortCode), zap.Error(err))
		return "", err
	}
	return url, nil
}

func (s *ShortLinkService) GetShortLinkByCode(shortCode string) (*models.ShortLink, error) {
	var shortLink models.ShortLink
	err := s.repo.GetByShortCode(&shortLink, shortCode)
	if err != nil {
		s.Log.Warn("Short link not found or inactive/expired", zap.String("shortCode", shortCode), zap.Error(err))
		return nil, err
	}
	return &shortLink, nil
}

func (s *ShortLinkService) GetLinksUser(userID uuid.UUID) ([]*models.ShortLink, error) {
	shortLinks, err := s.repo.GetByUserID(userID)
	if err != nil {
		s.Log.Warn("Failed to get short links for user", zap.String("userID", userID.String()), zap.Error(err))
		return nil, err
	}
	return shortLinks, nil
}

func (s *ShortLinkService) DeactivateShortLink(id, userID uuid.UUID) error {
	err := s.repo.DeactivateByID(id, userID)
	if err != nil {
		s.Log.Warn("Failed to deactivate short link", zap.String("id", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *ShortLinkService) GetByID(id string) (*models.ShortLink, error) {
	return s.repo.GetByID(id)
}
