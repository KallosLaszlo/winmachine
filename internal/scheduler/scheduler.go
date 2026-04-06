package scheduler

import (
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"winmachine/internal/backup"
	"winmachine/internal/config"
)

type Scheduler struct {
	cron   *cron.Cron
	engine *backup.Engine
	cfg    *config.Config
	mu     sync.Mutex
	paused bool
}

func New(cfg *config.Config, engine *backup.Engine) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		engine: engine,
		cfg:    cfg,
	}
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule := s.cfg.ScheduleInterval
	if schedule == "" {
		schedule = "@every 1h"
	}

	_, err := s.cron.AddFunc(schedule, func() {
		s.mu.Lock()
		paused := s.paused
		s.mu.Unlock()

		if paused {
			return
		}

		if err := s.engine.Run(); err != nil {
			log.Printf("scheduled backup failed: %v", err)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	log.Printf("scheduler started with interval: %s", schedule)
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("scheduler stopped")
}

func (s *Scheduler) RunNow() error {
	go func() {
		if err := s.engine.Run(); err != nil {
			log.Printf("manual backup failed: %v", err)
		}
	}()
	return nil
}

func (s *Scheduler) SetPaused(paused bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = paused
	if paused {
		log.Println("backups paused")
	} else {
		log.Println("backups resumed")
	}
}

func (s *Scheduler) IsPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paused
}
