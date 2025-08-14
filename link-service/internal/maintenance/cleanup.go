package maintenance

import (
	"context"
	"link-service/internal/repository"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	c         *cron.Cron
	log       *zap.Logger
	shortRepo *repository.ShortLinkRepository
	clickRepo *repository.ClickRepository
}

func NewScheduler(log *zap.Logger, shortRepo *repository.ShortLinkRepository, clickRepo *repository.ClickRepository) *Scheduler {
	// Используем cron с секундами отключёнными (стандартный 5-полюсный синтаксис) и локацией из системы.
	c := cron.New(cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow)), cron.WithChain())
	return &Scheduler{
		c: c, log: log,
		shortRepo: shortRepo,
		clickRepo: clickRepo,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.c.AddFunc("0 3 * * *", func() {
		s.cleanOldLinksAndClicks()
	})
	if err != nil {
		return err
	}
	s.c.Start()
	s.log.Info("Запущен планировщик")
	// Очистка при старте
	go s.cleanOldLinksAndClicks()

	go func() {
		<-ctx.Done()
		ctxStop := s.c.Stop()
		<-ctxStop.Done()
	}()
	return nil
}

func (s *Scheduler) cleanOldLinksAndClicks() {
	// Автоматическая деактивация ссылок
	if err := s.shortRepo.DeactivateExpiredAnonLinks(); err == nil {
		s.log.Info("Деактивированы истёкшие анонимные ссылки")
	}
	if err := s.shortRepo.DeactivateExpiredUserLinks(); err == nil {
		s.log.Info("Деактивированы истёкшие пользовательские ссылки")
	}
	s.log.Info("Запущена очистка старых ссылок и кликов")
	anonLinks, err := s.shortRepo.FindExpiredAnonLinks()
	if err == nil {
		for _, link := range anonLinks {
			s.clickRepo.DeleteClicksByShortLinkID(link.ID)
			s.shortRepo.DeleteLink(link)
			s.log.Info("Удалена анонимная истёкшая ссылка", zap.String("short_code", link.ShortCode))
		}
	}

	anonInactiveLinks, err := s.shortRepo.FindExpiredInactiveAnonLinks()
	if err == nil {
		for _, link := range anonInactiveLinks {
			s.clickRepo.DeleteClicksByShortLinkID(link.ID)
			s.shortRepo.DeleteLink(link)
			s.log.Info("Удалена анонимная деактивированная ссылка", zap.String("short_code", link.ShortCode))
		}
	}

	userLinks, err := s.shortRepo.FindExpiredInactiveUserLinks()
	if err == nil {
		for _, link := range userLinks {
			s.clickRepo.DeleteClicksByShortLinkID(link.ID)
			s.shortRepo.DeleteLink(link)
			s.log.Info("Удалена пользовательская деактивированная истёкшая ссылка", zap.String("short_code", link.ShortCode))
		}
	}
}
