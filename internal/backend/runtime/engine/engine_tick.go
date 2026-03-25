package engine

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/ports"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/session"
	"pause/internal/logx"
)

func (e *Engine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		logx.Infof("engine.start")
		e.mu.Lock()
		if e.lastTick.IsZero() {
			// Seed the baseline once at startup so the first scheduler tick accounts
			// for the first elapsed second instead of waiting an extra cycle.
			e.lastTick = time.Now()
			e.tickRemainder = 0
		}
		e.mu.Unlock()

		ticker := time.NewTicker(time.Second)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					e.Tick(now)
				}
			}
		}()
	})
}

func (e *Engine) Tick(now time.Time) {
	idleSec := e.idleProvider.CurrentIdleSeconds()
	locked := e.lockProvider.IsScreenLocked()

	e.mu.Lock()
	defer e.mu.Unlock()

	firstTick := e.lastTick.IsZero()
	if firstTick {
		e.lastTick = now
	}

	settings := e.store.Get()
	effectiveReminders := e.effectiveReminderConfigsLocked(e.reminders)
	wasLocked := e.currentLocked
	e.currentIdleSec = idleSec
	e.currentLocked = locked
	if !wasLocked && locked {
		logx.Infof(
			"engine.screen_locked timer_mode=%s idle_sec=%d threshold_sec=%d",
			settings.Timer.Mode,
			idleSec,
			settings.Timer.IdlePauseThresholdSec,
		)
	} else if wasLocked && !locked {
		logx.Infof(
			"engine.screen_unlocked timer_mode=%s idle_sec=%d threshold_sec=%d",
			settings.Timer.Mode,
			idleSec,
			settings.Timer.IdlePauseThresholdSec,
		)
	}
	e.lastTickActive = e.isTickActive(settings)
	rawDeltaSec := int(now.Sub(e.lastTick).Seconds())
	appliedDeltaSec := rawDeltaSec
	if firstTick {
		e.logTickLocked(now, settings, effectiveReminders, "bootstrap", 0, 0, nil)
		return
	}

	e.session.Tick(now)
	if view := e.session.CurrentView(now); view != nil && view.Status == string(session.StatusCompleted) {
		e.recordBreakCompletedLocked(view)
		logx.Infof(
			"break.completed reasons=%s duration_sec=%d",
			joinReasons(view.Reasons),
			int(view.EndsAt.Sub(view.StartedAt).Seconds()),
		)
		if err := e.soundPlayer.PlayBreakEnd(settings.Sound); err != nil {
			logx.Warnf("break.end_sound_err err=%v", err)
		}
	}
	e.session.ClearIfDone()

	if !settings.GlobalEnabled {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "global_disabled", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if e.session.IsActive() {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "session_active", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	if !e.lastTickActive {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "idle_paused", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	elapsed := now.Sub(e.lastTick) + e.tickRemainder
	if elapsed < 0 {
		e.lastTick = now
		e.tickRemainder = 0
		e.logTickLocked(now, settings, effectiveReminders, "negative_elapsed", rawDeltaSec, 0, nil)
		return
	}

	rawDeltaSec = int(elapsed / time.Second)
	if rawDeltaSec <= 0 {
		e.lastTick = now
		e.tickRemainder = elapsed
		e.logTickLocked(now, settings, effectiveReminders, "sub_second_elapsed", rawDeltaSec, 0, nil)
		return
	}

	appliedDeltaSec = rawDeltaSec
	e.tickRemainder = elapsed - (time.Duration(appliedDeltaSec) * time.Second)

	evt := e.scheduler.OnActiveSeconds(appliedDeltaSec, effectiveReminders)
	e.lastTick = now
	if evt == nil {
		e.logTickLocked(now, settings, effectiveReminders, "no_event", rawDeltaSec, appliedDeltaSec, nil)
		return
	}

	restEvent, notifyReminderIDs := splitReminderEventByType(evt, effectiveReminders)
	if len(notifyReminderIDs) > 0 {
		e.notifyRemindersLocked(notifyReminderIDs, settings.UI.Language)
	}
	if restEvent == nil {
		e.logTickLocked(now, settings, effectiveReminders, "notification_event", rawDeltaSec, appliedDeltaSec, evt)
		return
	}

	e.session.StartBreak(now, restEvent, settings.Enforcement.OverlaySkipAllowed)
	e.recordBreakStartedLocked(now, "scheduled", restEvent)
	logx.Infof(
		"break.started source=scheduled reasons=%s break_sec=%d skip_allowed=%t",
		joinReminderTypes(restEvent.Reasons),
		restEvent.BreakSec,
		settings.Enforcement.OverlaySkipAllowed,
	)
	e.logTickLocked(now, settings, effectiveReminders, "event", rawDeltaSec, appliedDeltaSec, restEvent)
}

func (e *Engine) isTickActive(cfg settings.Settings) bool {
	if cfg.Timer.Mode == settings.TimerModeRealTime {
		return true
	}
	if e.currentLocked {
		return false
	}
	return e.currentIdleSec < cfg.Timer.IdlePauseThresholdSec
}

func (e *Engine) logTickLocked(now time.Time, settings settings.Settings, reminders []reminder.Reminder, reason string, rawDeltaSec, appliedDeltaSec int, evt *scheduler.Event) {
	sessionStatus := "none"
	if view := e.session.CurrentView(now); view != nil {
		sessionStatus = view.Status
	}
	nextByID := e.scheduler.NextByID(reminders)
	nextSummary := "none"
	if len(nextByID) > 0 {
		parts := make([]string, 0, len(nextByID))
		for id, next := range nextByID {
			parts = append(parts, strconv.FormatInt(id, 10)+"="+strconv.Itoa(next))
		}
		sort.Strings(parts)
		nextSummary = strings.Join(parts, ",")
	}

	evtReasons := ""
	evtBreak := 0
	if evt != nil {
		evtBreak = evt.BreakSec
		reasons := make([]string, 0, len(evt.Reasons))
		for _, r := range evt.Reasons {
			reasons = append(reasons, strconv.FormatInt(int64(r), 10))
		}
		evtReasons = strings.Join(reasons, "+")
	}

	logx.Debugf(
		"engine.tick reason=%s now_unix=%d raw_delta=%d applied_delta=%d idle_sec=%d tick_active=%t session=%s next=%s evt_reasons=%s evt_break=%d",
		reason,
		now.Unix(),
		rawDeltaSec,
		appliedDeltaSec,
		e.currentIdleSec,
		e.lastTickActive,
		sessionStatus,
		nextSummary,
		evtReasons,
		evtBreak,
	)
}

func (e *Engine) notifyRemindersLocked(reminderIDs []int64, language string) {
	if len(reminderIDs) == 0 || e.notifier == nil {
		return
	}
	names := make([]string, 0, len(reminderIDs))
	byID := make(map[int64]reminder.Reminder, len(e.reminders))
	for _, reminder := range e.reminders {
		byID[reminder.ID] = reminder
	}
	for _, id := range reminderIDs {
		reminder, ok := byID[id]
		if !ok {
			continue
		}
		name := strings.TrimSpace(reminder.Name)
		if name == "" {
			name = strconv.FormatInt(reminder.ID, 10)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return
	}

	title := "Reminder"
	body := strings.Join(names, " · ")
	if language == settings.UILanguageZhCN {
		title = "提醒"
	}
	notifier := e.notifier
	keyParts := make([]string, 0, len(reminderIDs))
	for _, id := range reminderIDs {
		keyParts = append(keyParts, strconv.FormatInt(id, 10))
	}
	reminderKey := strings.Join(keyParts, "+")
	go func(n ports.Notifier, t string, b string, key string) {
		if err := n.ShowReminder(t, b); err != nil {
			logx.Warnf("reminder.notification_err reminders=%s err=%v", key, err)
			return
		}
		logx.Infof("reminder.notification_sent reminders=%s", key)
	}(notifier, title, body, reminderKey)
}
