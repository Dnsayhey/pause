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
		{name: "auto follows system light", setting: "auto", system: "light", want: "light"},
		{name: "auto follows system dark", setting: "auto", system: "dark", want: "dark"},
		{name: "auto fallback dark when unknown", setting: "auto", system: "", want: "dark"},
		{name: "invalid setting fallback dark", setting: "sepia", system: "unknown", want: "dark"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveEffectiveThemeWithSystem(tc.setting, tc.system)
			if got != tc.want {
				t.Fatalf("resolveEffectiveThemeWithSystem(%q, %q) = %q, want %q", tc.setting, tc.system, got, tc.want)
			}
		})
	}
}
