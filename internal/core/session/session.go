package session

import (
	"errors"
	"time"

	"pause/internal/core/config"
	"pause/internal/core/scheduler"
)

type Status string

const (
	StatusIdle      Status = "idle"
	StatusReminding Status = "reminding"
	StatusResting   Status = "resting"
	StatusCompleted Status = "completed"
	StatusSkipped   Status = "skipped"
)

type Session struct {
	status    Status
	reasons   []int64
	startedAt time.Time
	endsAt    time.Time
	canSkip   bool
}

type Manager struct {
	current *Session
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) IsActive() bool {
	return m.current != nil && (m.current.status == StatusReminding || m.current.status == StatusResting)
}

func (m *Manager) StartBreak(now time.Time, evt *scheduler.Event, canSkip bool) {
	if evt == nil {
		return
	}

	reasons := make([]int64, 0, len(evt.Reasons))
	for _, reason := range evt.Reasons {
		reasons = append(reasons, int64(reason))
	}

	m.current = &Session{
		status:    StatusReminding,
		reasons:   reasons,
		startedAt: now,
		endsAt:    now.Add(time.Duration(evt.BreakSec) * time.Second),
		canSkip:   canSkip,
	}

	// v1 has no pre-break countdown; session enters rest immediately.
	m.current.status = StatusResting
}

func (m *Manager) Tick(now time.Time) {
	if m.current == nil {
		return
	}
	if now.After(m.current.endsAt) || now.Equal(m.current.endsAt) {
		m.current.status = StatusCompleted
	}
}

func (m *Manager) ClearIfDone() {
	if m.current == nil {
		return
	}
	if m.current.status == StatusCompleted || m.current.status == StatusSkipped {
		m.current = nil
	}
}

func (m *Manager) Skip() error {
	if m.current == nil {
		return errors.New("no active break")
	}
	if !m.current.canSkip {
		return errors.New("skip not allowed")
	}
	m.current.status = StatusSkipped
	return nil
}

func (m *Manager) SetCanSkip(canSkip bool) {
	if m.current == nil {
		return
	}
	m.current.canSkip = canSkip
}

func (m *Manager) CurrentView(now time.Time) *config.BreakSessionView {
	if m.current == nil {
		return nil
	}

	remaining := int(m.current.endsAt.Sub(now).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	return &config.BreakSessionView{
		Status:       string(m.current.status),
		Reasons:      append([]int64(nil), m.current.reasons...),
		StartedAt:    m.current.startedAt,
		EndsAt:       m.current.endsAt,
		RemainingSec: remaining,
		CanSkip:      m.current.canSkip,
	}
}
