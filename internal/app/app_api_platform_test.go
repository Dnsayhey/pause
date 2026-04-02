package app

import "testing"

func TestNormalizePlatformOS(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "darwin", want: "macos"},
		{input: "windows", want: "windows"},
		{input: "linux", want: "linux"},
		{input: "freebsd", want: "unknown"},
	}

	for _, tc := range cases {
		if got := normalizePlatformOS(tc.input); got != tc.want {
			t.Fatalf("normalizePlatformOS(%q)=%q want=%q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizePlatformArch(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "amd64", want: "x64"},
		{input: "arm64", want: "arm64"},
		{input: "386", want: "x86"},
		{input: "arm", want: "unknown"},
	}

	for _, tc := range cases {
		if got := normalizePlatformArch(tc.input); got != tc.want {
			t.Fatalf("normalizePlatformArch(%q)=%q want=%q", tc.input, got, tc.want)
		}
	}
}
