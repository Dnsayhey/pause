package engine

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/state"
)

func buildImmediateBreakEvent(reminders []reminder.Reminder, nextByID map[int64]int, forcedReason int64) *scheduler.Event {
	reasonKey := normalizeReminderID(forcedReason)
	if reasonKey <= 0 {
		reasonKey = selectImmediateReason(nextByID)
	}
	if reasonKey <= 0 {
		return nil
	}
	cfg, ok := findReminderByID(reminders, reasonKey)
	if !ok || !cfg.Enabled || cfg.BreakSec <= 0 || !isRestReminderType(cfg.ReminderType) {
		return nil
	}
	return &scheduler.Event{
		Reasons:  []scheduler.ReminderType{scheduler.ReminderType(reasonKey)},
		BreakSec: cfg.BreakSec,
	}
}

func normalizeReminderID(id int64) int64 {
	if id <= 0 {
		return 0
	}
	return id
}

func cloneReminderConfigs(reminders []reminder.Reminder) []reminder.Reminder {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]reminder.Reminder, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}

func findReminderByID(reminders []reminder.Reminder, id int64) (reminder.Reminder, bool) {
	norm := normalizeReminderID(id)
	for _, cfg := range reminders {
		if cfg.ID == norm {
			return cfg, true
		}
	}
	return reminder.Reminder{}, false
}

func selectImmediateReason(nextByID map[int64]int) int64 {
	var bestID int64
	bestNext := -1
	for id, next := range nextByID {
		if next < 0 {
			continue
		}
		if bestNext < 0 || next < bestNext || (next == bestNext && (bestID == 0 || id < bestID)) {
			bestID = id
			bestNext = next
		}
	}
	return bestID
}

func resetSchedulerByReasons(s *scheduler.Scheduler, reasons []scheduler.ReminderType) {
	if s == nil || len(reasons) == 0 {
		return
	}

	seen := map[scheduler.ReminderType]struct{}{}
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		s.ResetByID(int64(reason))
	}
}

func joinReminderTypes(reasons []scheduler.ReminderType) string {
	if len(reasons) == 0 {
		return "none"
	}

	labels := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		labels = append(labels, strconv.FormatInt(int64(reason), 10))
	}
	return strings.Join(labels, "+")
}

func joinReasons(reasons []int64) string {
	if len(reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, strconv.FormatInt(reason, 10))
	}
	return strings.Join(parts, "+")
}

func marshalPatchForLog(patch settings.SettingsPatch) string {
	raw, err := json.Marshal(patch)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func nextReasons(reminders []state.ReminderRuntime, defs []reminder.Reminder) []int64 {
	restReminderIDs := map[int64]struct{}{}
	for _, def := range defs {
		if !isRestReminderType(def.ReminderType) {
			continue
		}
		restReminderIDs[def.ID] = struct{}{}
	}

	minNext := -1
	for _, reminder := range reminders {
		if _, isRest := restReminderIDs[reminder.ID]; !isRest {
			continue
		}
		if !reminder.Enabled || reminder.Paused || reminder.NextInSec < 0 {
			continue
		}
		if minNext < 0 || reminder.NextInSec < minNext {
			minNext = reminder.NextInSec
		}
	}
	if minNext < 0 {
		return []int64{}
	}

	reasons := make([]int64, 0, len(reminders))
	for _, reminder := range reminders {
		if _, isRest := restReminderIDs[reminder.ID]; !isRest {
			continue
		}
		if !reminder.Enabled || reminder.Paused || reminder.NextInSec < 0 {
			continue
		}
		if reminder.NextInSec-minNext <= scheduler.MergeWindowSec {
			reasons = append(reasons, reminder.ID)
		}
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return reasons
}

func isRestReminderType(reminderType string) bool {
	return reminder.IsRestReminderType(reminderType)
}

func splitReminderEventByType(evt *scheduler.Event, reminders []reminder.Reminder) (*scheduler.Event, []int64) {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil, nil
	}

	byID := make(map[int64]reminder.Reminder, len(reminders))
	for _, reminder := range reminders {
		byID[reminder.ID] = reminder
	}

	restReasons := make([]scheduler.ReminderType, 0, len(evt.Reasons))
	notifyReminderIDs := make([]int64, 0, len(evt.Reasons))
	restBreakSec := 0

	for _, reason := range evt.Reasons {
		id := normalizeReminderID(int64(reason))
		reminder, ok := byID[id]
		if !ok {
			restReasons = append(restReasons, reason)
			if evt.BreakSec > restBreakSec {
				restBreakSec = evt.BreakSec
			}
			continue
		}
		if isRestReminderType(reminder.ReminderType) {
			restReasons = append(restReasons, reason)
			if reminder.BreakSec > restBreakSec {
				restBreakSec = reminder.BreakSec
			}
			continue
		}
		notifyReminderIDs = append(notifyReminderIDs, id)
	}

	sort.Slice(notifyReminderIDs, func(i, j int) bool { return notifyReminderIDs[i] < notifyReminderIDs[j] })
	if len(notifyReminderIDs) > 1 {
		uniq := notifyReminderIDs[:0]
		var last int64
		lastSet := false
		for _, id := range notifyReminderIDs {
			if lastSet && id == last {
				continue
			}
			uniq = append(uniq, id)
			last = id
			lastSet = true
		}
		notifyReminderIDs = uniq
	}

	if len(restReasons) == 0 {
		return nil, notifyReminderIDs
	}
	if restBreakSec <= 0 {
		restBreakSec = evt.BreakSec
		if restBreakSec <= 0 {
			restBreakSec = 1
		}
	}

	return &scheduler.Event{
		Reasons:  restReasons,
		BreakSec: restBreakSec,
	}, notifyReminderIDs
}
