package ports

type LockStateProvider interface {
	IsScreenLocked() bool
}
