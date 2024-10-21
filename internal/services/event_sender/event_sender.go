package eventsender

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	"github.com/google/uuid"
)

type EventPublisher interface {
	Publish(ctx context.Context, key, data []byte) error
}

type EventProvider interface {
	NewEvents(ctx context.Context, limit int) ([]models.Event, error)
	SetEventDone(ctx context.Context, eventId uuid.UUID) (models.Event, error)
}

type Sender struct {
	log            *slog.Logger
	eventPublisher EventPublisher
	eventProvider  EventProvider
	stopChan       chan struct{}
}

func NewSender(
	log *slog.Logger,
	eventPublisher EventPublisher,
	eventProvider EventProvider,
) *Sender {
	return &Sender{log: log, eventPublisher: eventPublisher, eventProvider: eventProvider}
}

func (s *Sender) StartProducing(ctx context.Context, limit int, interval time.Duration) {
	const op = "service.event_sender.StartProducing"
	log := s.log.With(slog.String("op", op))

	ticker := time.NewTicker(interval)

	log.Info("starting producing events", slog.Int("limit", limit), slog.Duration("interval", interval))

	if err := ctx.Err(); err != nil {
		log.Info("stopping event producing", sl.Err(err))
		return
	}

	go func() {
		defer func() {
			log.Info("stopping event producing")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				events, err := s.eventProvider.NewEvents(ctx, limit)
				if err != nil {
					log.Error("failed to get new events", sl.Err(err))
					continue
				}

				wg := &sync.WaitGroup{}
				for _, event := range events {
					wg.Add(1)
					go s.processEvent(ctx, wg, event)
				}
				wg.Wait()
			}
		}
	}()
}

func (s *Sender) processEvent(ctx context.Context, wg *sync.WaitGroup, event models.Event) {
	const op = "service.event_sender.processEvent"
	log := s.log.With(slog.String("op", op))

	defer wg.Done()

	if err := s.eventPublisher.Publish(ctx, []byte(event.Type), []byte(event.Payload)); err != nil {
		log.Error("failed to Publish event", sl.Err(err))
		return
	}

	if _, err := s.eventProvider.SetEventDone(ctx, event.ID); err != nil {
		log.Error("failed to mark event as done", slog.String("eventId", event.ID.String()), sl.Err(err))
		return
	}
}

func (s *Sender) StopSending() {
	s.stopChan <- struct{}{}
}
