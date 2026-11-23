package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// safeNotificationService wraps Wails's NotificationService so startup errors on
// restricted systems degrade gracefully instead of aborting application launch.
type safeNotificationService struct {
	ns        *notifications.NotificationService
	mu        sync.RWMutex
	available bool
	seq       atomic.Uint64
}

func newSafeNotificationService(onActivate func()) *safeNotificationService {
	ns := notifications.New()
	ns.OnNotificationResponse(func(_ notifications.NotificationResult) {
		if onActivate != nil {
			onActivate()
		}
	})
	return &safeNotificationService{ns: ns}
}

func (s *safeNotificationService) ServiceName() string {
	return "safeNotificationService"
}

func (s *safeNotificationService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	if s.ns == nil {
		return nil
	}
	if err := s.ns.ServiceStartup(ctx, options); err != nil {
		log.Printf("notifications unavailable: %v", err)
		return nil
	}
	s.mu.Lock()
	s.available = true
	s.mu.Unlock()
	return nil
}

func (s *safeNotificationService) ServiceShutdown() error {
	s.mu.RLock()
	ok := s.available && s.ns != nil
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	return s.ns.ServiceShutdown()
}

func (s *safeNotificationService) send(title, body string) {
	if s == nil {
		return
	}
	s.mu.RLock()
	ok := s.available && s.ns != nil
	s.mu.RUnlock()
	if !ok || title == "" {
		return
	}

	id := fmt.Sprintf("s7-trend-%d", s.seq.Add(1))
	go func() {
		if err := s.ns.SendNotification(notifications.NotificationOptions{
			ID:    id,
			Title: title,
			Body:  body,
		}); err != nil {
			log.Printf("notification send failed: %v", err)
		}
	}()
}
