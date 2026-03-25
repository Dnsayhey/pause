package ports

type Notifier interface {
	ShowReminder(title, body string) error
}
