package meta

import (
	_ "embed"
	"strings"
)

// AppBundleID can be injected at build time:
//
//	-ldflags "-X pause/internal/meta.AppBundleID=com.example.pause"
var AppBundleID string

//go:embed bundle_id.txt
var defaultBundleID string

func EffectiveAppBundleID() string {
	if v := strings.TrimSpace(AppBundleID); v != "" {
		return v
	}
	if v := strings.TrimSpace(defaultBundleID); v != "" {
		return v
	}
	return "pause.invalid.bundle-id"
}

func EffectiveLoginHelperBundleID() string {
	return EffectiveAppBundleID() + ".loginhelper"
}

func SingleInstanceID() string {
	return EffectiveAppBundleID() + ".single-instance"
}
