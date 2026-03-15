package desktop

type StatusBarLocaleStrings struct {
	PopoverTitle          string
	BreakNowButton        string
	PauseButton           string
	ResumeButton          string
	OpenAppButton         string
	AboutMenuItem         string
	QuitMenuItem          string
	MoreButtonTip         string
	Tooltip               string
	StatusLineFallback    string
	NextBreakLineFallback string
}

const (
	StatusBarActionBreakNow   = 1
	StatusBarActionPause      = 2
	StatusBarActionResume     = 4
	StatusBarActionOpenWindow = 5
	StatusBarActionQuit       = 6

	StatusBarActionPauseReminderBase  = 1000
	StatusBarActionResumeReminderBase = 2000
)

type StatusBarController interface {
	Init(onAction func(actionID int))
	Update(status, countdown, title string, paused bool, progress float64, remindersPayload string)
	SetLocale(strings StatusBarLocaleStrings)
	Destroy()
}

type BreakOverlayController interface {
	Init(onSkip func())
	Show(allowSkip bool, skipButtonTitle string, countdownText string, theme string) bool
	Hide()
	Destroy()
	IsNative() bool
}
