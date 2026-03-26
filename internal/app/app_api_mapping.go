package app

import (
	"strings"

	analyticsdomain "pause/internal/backend/domain/analytics"
	reminderdomain "pause/internal/backend/domain/reminder"
	settingsdomain "pause/internal/backend/domain/settings"
	runtimestate "pause/internal/backend/runtime/state"
)

func reminderCreateInputToDomain(input ReminderCreateInput) reminderdomain.CreateInput {
	return reminderdomain.CreateInput{
		Name:         input.Name,
		IntervalSec:  input.IntervalSec,
		BreakSec:     input.BreakSec,
		Enabled:      input.Enabled,
		ReminderType: input.ReminderType,
	}
}

func reminderPatchToDomain(patch ReminderPatch) reminderdomain.Patch {
	return reminderdomain.Patch{
		ID:           patch.ID,
		Name:         patch.Name,
		Enabled:      patch.Enabled,
		IntervalSec:  patch.IntervalSec,
		BreakSec:     patch.BreakSec,
		ReminderType: patch.ReminderType,
	}
}

func settingsFromDomain(source settingsdomain.Settings) Settings {
	return Settings{
		Enforcement: EnforcementSettings{
			OverlaySkipAllowed: source.Enforcement.OverlaySkipAllowed,
		},
		Sound: SoundSettings{
			Enabled: source.Sound.Enabled,
			Volume:  source.Sound.Volume,
		},
		Timer: TimerSettings{
			Mode:                  source.Timer.Mode,
			IdlePauseThresholdSec: source.Timer.IdlePauseThresholdSec,
		},
		UI: UISettings{
			ShowTrayCountdown: source.UI.ShowTrayCountdown,
			Language:          source.UI.Language,
			Theme:             source.UI.Theme,
		},
	}
}

func settingsPatchToDomain(patch SettingsPatch) settingsdomain.SettingsPatch {
	result := settingsdomain.SettingsPatch{}
	if patch.Enforcement != nil {
		result.Enforcement = &settingsdomain.EnforcementSettingsPatch{
			OverlaySkipAllowed: patch.Enforcement.OverlaySkipAllowed,
		}
	}
	if patch.Sound != nil {
		result.Sound = &settingsdomain.SoundSettingsPatch{
			Enabled: patch.Sound.Enabled,
			Volume:  patch.Sound.Volume,
		}
	}
	if patch.Timer != nil {
		result.Timer = &settingsdomain.TimerSettingsPatch{
			Mode:                  patch.Timer.Mode,
			IdlePauseThresholdSec: patch.Timer.IdlePauseThresholdSec,
		}
	}
	if patch.UI != nil {
		result.UI = &settingsdomain.UISettingsPatch{
			ShowTrayCountdown: patch.UI.ShowTrayCountdown,
			Language:          patch.UI.Language,
			Theme:             patch.UI.Theme,
		}
	}
	return result
}

func runtimeStateFromDomain(source runtimestate.RuntimeState) RuntimeState {
	var currentSession *BreakSessionView
	if source.CurrentSession != nil {
		currentSession = &BreakSessionView{
			Status:       source.CurrentSession.Status,
			Reasons:      cloneInt64Slice(source.CurrentSession.Reasons),
			StartedAt:    source.CurrentSession.StartedAt,
			EndsAt:       source.CurrentSession.EndsAt,
			RemainingSec: source.CurrentSession.RemainingSec,
			CanSkip:      source.CurrentSession.CanSkip,
		}
	}

	reminders := make([]ReminderRuntime, 0, len(source.Reminders))
	for _, reminder := range source.Reminders {
		reminders = append(reminders, ReminderRuntime{
			ID:           reminder.ID,
			Name:         strings.TrimSpace(reminder.Name),
			ReminderType: strings.TrimSpace(reminder.ReminderType),
			Enabled:      reminder.Enabled,
			Paused:       reminder.Paused,
			NextInSec:    reminder.NextInSec,
			IntervalSec:  reminder.IntervalSec,
			BreakSec:     reminder.BreakSec,
		})
	}

	return RuntimeState{
		Now:                source.Now,
		CurrentSession:     currentSession,
		Reminders:          reminders,
		NextBreakReason:    cloneInt64Slice(source.NextBreakReason),
		GlobalEnabled:      source.GlobalEnabled,
		TimerMode:          source.TimerMode,
		IdleThresholdSec:   source.IdleThresholdSec,
		LastTickActive:     source.LastTickActive,
		CurrentIdleSec:     source.CurrentIdleSec,
		ShowTrayCountdown:  source.ShowTrayCountdown,
		OverlaySkipAllowed: source.OverlaySkipAllowed,
		OverlayNative:      source.OverlayNative,
		EffectiveLanguage:  source.EffectiveLanguage,
		EffectiveTheme:     source.EffectiveTheme,
	}
}

func analyticsWeeklyStatsFromDomain(source analyticsdomain.WeeklyStats) AnalyticsWeeklyStats {
	reminders := make([]AnalyticsReminderStat, 0, len(source.Reminders))
	for _, reminder := range source.Reminders {
		reminders = append(reminders, analyticsReminderStatFromDomain(reminder))
	}
	return AnalyticsWeeklyStats{
		FromSec:   source.FromSec,
		ToSec:     source.ToSec,
		Reminders: reminders,
		Summary: AnalyticsSummaryStats{
			TotalSessions:       source.Summary.TotalSessions,
			TotalCompleted:      source.Summary.TotalCompleted,
			TotalSkipped:        source.Summary.TotalSkipped,
			TotalCanceled:       source.Summary.TotalCanceled,
			TotalActualBreakSec: source.Summary.TotalActualBreakSec,
			AvgActualBreakSec:   source.Summary.AvgActualBreakSec,
		},
	}
}

func analyticsSummaryFromDomain(source analyticsdomain.Summary) AnalyticsSummary {
	return AnalyticsSummary{
		FromSec:             source.FromSec,
		ToSec:               source.ToSec,
		TotalSessions:       source.TotalSessions,
		TotalCompleted:      source.TotalCompleted,
		TotalSkipped:        source.TotalSkipped,
		TotalCanceled:       source.TotalCanceled,
		CompletionRate:      source.CompletionRate,
		SkipRate:            source.SkipRate,
		TotalActualBreakSec: source.TotalActualBreakSec,
		AvgActualBreakSec:   source.AvgActualBreakSec,
	}
}

func analyticsTrendFromDomain(source analyticsdomain.Trend) AnalyticsTrend {
	points := make([]AnalyticsTrendPoint, 0, len(source.Points))
	for _, point := range source.Points {
		points = append(points, AnalyticsTrendPoint{
			Day:                 point.Day,
			TotalSessions:       point.TotalSessions,
			TotalCompleted:      point.TotalCompleted,
			TotalSkipped:        point.TotalSkipped,
			TotalCanceled:       point.TotalCanceled,
			CompletionRate:      point.CompletionRate,
			SkipRate:            point.SkipRate,
			TotalActualBreakSec: point.TotalActualBreakSec,
			AvgActualBreakSec:   point.AvgActualBreakSec,
		})
	}
	return AnalyticsTrend{
		FromSec: source.FromSec,
		ToSec:   source.ToSec,
		Points:  points,
	}
}

func analyticsBreakTypeDistributionFromDomain(source analyticsdomain.BreakTypeDistribution) AnalyticsBreakTypeDistribution {
	items := make([]AnalyticsBreakTypeDistributionItem, 0, len(source.Items))
	for _, item := range source.Items {
		items = append(items, AnalyticsBreakTypeDistributionItem{
			ReminderID:      item.ReminderID,
			ReminderName:    item.ReminderName,
			TriggeredCount:  item.TriggeredCount,
			CompletedCount:  item.CompletedCount,
			SkippedCount:    item.SkippedCount,
			CanceledCount:   item.CanceledCount,
			CompletionRate:  item.CompletionRate,
			SkipRate:        item.SkipRate,
			TriggeredShare:  item.TriggeredShare,
			ReminderType:    item.ReminderType,
			ReminderEnabled: item.ReminderEnabled,
		})
	}
	return AnalyticsBreakTypeDistribution{
		FromSec:        source.FromSec,
		ToSec:          source.ToSec,
		TotalTriggered: source.TotalTriggered,
		Items:          items,
	}
}

func analyticsReminderStatFromDomain(source analyticsdomain.ReminderStat) AnalyticsReminderStat {
	return AnalyticsReminderStat{
		ReminderID:          source.ReminderID,
		ReminderName:        source.ReminderName,
		Enabled:             source.Enabled,
		ReminderType:        source.ReminderType,
		TriggeredCount:      source.TriggeredCount,
		CompletedCount:      source.CompletedCount,
		SkippedCount:        source.SkippedCount,
		CanceledCount:       source.CanceledCount,
		TotalActualBreakSec: source.TotalActualBreakSec,
		AvgActualBreakSec:   source.AvgActualBreakSec,
	}
}

func cloneInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]int64, 0, len(values))
	cloned = append(cloned, values...)
	return cloned
}
