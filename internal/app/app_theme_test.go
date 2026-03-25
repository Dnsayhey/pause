package app

import "testing"

func TestResolveEffectiveThemeWithSystem(t *testing.T) {
	cases := []struct {
		name    string
		setting string
		system  string
		want    string
	}{
		{name: "explicit light", setting: "light", system: "dark", want: "light"},
		{name: "explicit dark", setting: "dark", system: "light", want: "dark"},
		{name: "auto -> light", setting: "auto", system: "light", want: "light"},
		{name: "auto -> dark", setting: "auto", system: "dark", want: "dark"},
		{name: "fallback dark", setting: "auto", system: "unknown", want: "dark"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveEffectiveThemeWithSystem(tc.setting, tc.system); got != tc.want {
				t.Fatalf("resolveEffectiveThemeWithSystem(%q,%q)=%q want=%q", tc.setting, tc.system, got, tc.want)
			}
		})
	}
}
