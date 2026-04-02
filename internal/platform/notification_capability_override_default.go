//go:build !dev

package platform

func notificationCapabilityDisabledByBuild() (string, bool) {
	return "", false
}
