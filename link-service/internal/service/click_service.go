package service

import (
	"encoding/json"
	"link-service/internal/models"
	"link-service/internal/repository"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ClickService struct {
	repo *repository.ClickRepository
	log  *zap.Logger
}

func NewClickService(repo *repository.ClickRepository, log *zap.Logger) *ClickService {
	return &ClickService{
		repo: repo,
		log:  log,
	}
}

func (s *ClickService) CreateClick(shortLinkID uuid.UUID, ip, userAgent string) error {
	click := &models.Click{
		ShortLinkID: shortLinkID,
		IP:          ip,
		UserAgent:   userAgent,
		ClickedAt:   time.Now(),
	}
	resp, err := http.Get("http://ip-api.com/json/" + ip)
	if err == nil {
		defer resp.Body.Close()
		var data struct {
			Country    string `json:"country"`
			RegionName string `json:"regionName"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
			click.Country = data.Country
			click.Region = data.RegionName
		}
	}

	return s.repo.Create(click)
}

type Stats struct {
	Total          int64
	UniqueIPCount  int64
	UniqueIPs      []string
	CountriesCount int
	Countries      []string
	CountriesStats map[string]int64
	DailyStats     map[string]int64
}

func (s *ClickService) GetStats(shortLinkID string) (Stats, error) {
	var stats Stats
	// Общее количество переходов
	total, err := s.repo.GetCount(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.Total = total

	// Уникальные IP
	uniqueIPCount, err := s.repo.GetUniqueIPCount(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.UniqueIPCount = uniqueIPCount

	uniqueIPs, err := s.repo.GetUniqueIPs(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.UniqueIPs = uniqueIPs

	// География по странам
	countries, err := s.repo.GetUniqueCountries(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.CountriesCount = len(countries)
	stats.Countries = countries

	countryStats, err := s.repo.GetCountryStats(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.CountriesStats = countryStats

	// График по дням
	dailyStats, err := s.repo.GetDailyStats(shortLinkID)
	if err != nil {
		return stats, err
	}
	stats.DailyStats = dailyStats

	return stats, nil
}

func (s *ClickService) GetClicks(shortLinkID string) ([]models.Click, error) {
	clicks, err := s.repo.GetClicksByShortLinkID(shortLinkID)
	if err != nil {
		return nil, err
	}
	sort.Slice(clicks, func(i, j int) bool { return clicks[i].ClickedAt.After(clicks[j].ClickedAt) })
	return clicks, nil
}
