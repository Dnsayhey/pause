package reminder

import "pause/internal/core/config"

func NormalizeID(id int64) int64 {
	if id <= 0 {
		return 0
	}
	return id
}

func CloneConfigs(reminders []config.ReminderConfig) []config.ReminderConfig {
	if len(reminders) == 0 {
		return nil
	}
	cloned := make([]config.ReminderConfig, 0, len(reminders))
	cloned = append(cloned, reminders...)
	return cloned
}

func FindByID(reminders []config.ReminderConfig, id int64) (config.ReminderConfig, bool) {
	norm := NormalizeID(id)
	for _, reminder := range reminders {
		if reminder.ID == norm {
			return reminder, true
		}
	}
	return config.ReminderConfig{}, false
}

func ApplyPatches(reminders []config.ReminderConfig, patches []config.ReminderPatch) []config.ReminderConfig {
	updated := CloneConfigs(reminders)
	for _, patch := range patches {
		updated = applyPatch(updated, patch)
	}
	return updated
}

func applyPatch(reminders []config.ReminderConfig, patch config.ReminderPatch) []config.ReminderConfig {
	id := NormalizeID(patch.ID)
	if id <= 0 {
		return reminders
	}

	idx := -1
	for i, reminder := range reminders {
		if reminder.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return reminders
	}

	if patch.Name != nil {
		reminders[idx].Name = *patch.Name
	}
	if patch.Enabled != nil {
		reminders[idx].Enabled = *patch.Enabled
	}
	if patch.IntervalSec != nil {
		reminders[idx].IntervalSec = *patch.IntervalSec
	}
	if patch.BreakSec != nil {
		reminders[idx].BreakSec = *patch.BreakSec
	}
	if patch.ReminderType != nil {
		reminders[idx].ReminderType = *patch.ReminderType
	}
	return reminders
}
