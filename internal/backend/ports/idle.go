package ports

type IdleProvider interface {
	CurrentIdleSeconds() int
}
