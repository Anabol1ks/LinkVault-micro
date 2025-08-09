package maintenance

import (
	"context"
	"time"

	"linkv-auth/internal/repository"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	c      *cron.Cron
	log    *zap.Logger
	rtRepo *repository.RefreshTokenRepository
}

func NewScheduler(log *zap.Logger, rtRepo *repository.RefreshTokenRepository) *Scheduler {
	// Используем cron с секундами отключёнными (стандартный 5-полюсный синтаксис) и локацией из системы.
	c := cron.New(cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow)), cron.WithChain())
	return &Scheduler{c: c, log: log, rtRepo: rtRepo}
}

func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.c.AddFunc("0 3 * * *", func() {
		s.cleanupExpired()
	})
	if err != nil {
		return err
	}
	// Запускаем
	s.c.Start()

	go func() {
		<-ctx.Done()
		ctxStop := s.c.Stop()
		<-ctxStop.Done()
	}()
	return nil
}

func (s *Scheduler) cleanupExpired() {
	now := time.Now()
	deleted, err := s.rtRepo.DeleteExpired(now)
	if err != nil {
		s.log.Error("Ошибка очистки просроченных refresh токенов", zap.Error(err))
		return
	}
	if deleted > 0 {
		s.log.Info("Удалены просроченные refresh токены", zap.Int64("count", deleted))
	}
}
