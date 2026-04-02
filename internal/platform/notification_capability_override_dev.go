//go:build dev

package platform

func notificationCapabilityDisabledByBuild() (string, bool) {
	return "notification capability disabled in dev build", true
}
