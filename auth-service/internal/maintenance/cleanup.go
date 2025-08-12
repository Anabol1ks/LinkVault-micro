package maintenance

import (
	"context"
	"time"

	"auth-service/internal/repository"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	c                 *cron.Cron
	log               *zap.Logger
	rtRepo            *repository.RefreshTokenRepository
	emailTokenRepo    *repository.EmailVerificationTokenRepository
	passwordResetRepo *repository.PasswordResetTokenRepository
}

func NewScheduler(log *zap.Logger, rtRepo *repository.RefreshTokenRepository, emailTokenRepo *repository.EmailVerificationTokenRepository, passwordResetRepo *repository.PasswordResetTokenRepository) *Scheduler {
	// Используем cron с секундами отключёнными (стандартный 5-полюсный синтаксис) и локацией из системы.
	c := cron.New(cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow)), cron.WithChain())
	return &Scheduler{c: c, log: log, rtRepo: rtRepo, emailTokenRepo: emailTokenRepo, passwordResetRepo: passwordResetRepo}
}

func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.c.AddFunc("0 3 * * *", func() {
		s.cleanupExpired()
	})
	if err != nil {
		return err
	}
	// Запускаем cron
	s.c.Start()

	// Очистка при старте
	go s.cleanupExpired()

	go func() {
		<-ctx.Done()
		ctxStop := s.c.Stop()
		<-ctxStop.Done()
	}()
	return nil
}

func (s *Scheduler) cleanupExpired() {
	now := time.Now()
	// Очистка refresh токенов
	if s.rtRepo != nil {
		deleted, err := s.rtRepo.DeleteExpired(now)
		if err != nil {
			s.log.Error("Ошибка очистки просроченных refresh токенов", zap.Error(err))
		} else if deleted > 0 {
			s.log.Info("Удалены просроченные refresh токены", zap.Int64("count", deleted))
		}
	}
	// Очистка email verification токенов
	if s.emailTokenRepo != nil {
		deleted, err := s.emailTokenRepo.DeleteExpired(now)
		if err != nil {
			s.log.Error("Ошибка очистки просроченных email verification токенов", zap.Error(err))
		} else if deleted > 0 {
			s.log.Info("Удалены просроченные email verification токены", zap.Int64("count", deleted))
		}
	}
	// Очистка password reset токенов
	if s.passwordResetRepo != nil {
		deleted, err := s.passwordResetRepo.DeleteExpired(now)
		if err != nil {
			s.log.Error("Ошибка очистки просроченных password reset токенов", zap.Error(err))
		} else if deleted > 0 {
			s.log.Info("Удалены просроченные password reset токены", zap.Int64("count", deleted))
		}
	}
}
