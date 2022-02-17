package p2p

import (
	"testing"
)

type sanitizeVersionCase struct {
	version string
	ref     string
	want    string
}

func Test_sanitizeVersion(t *testing.T) {
	cases := []sanitizeVersionCase{
		{version: "v1.0.0", ref: "v1.0.0", want: "v1.0.0"},
		{version: "v1.0.0-foo", ref: "v1.0.0", want: "v1.0.0"},
		{version: "v1.0.0-foo", ref: "v1.0.0-bar", want: "v1.0.0"},
		{version: "v6.0.0-foo", ref: "v1.0.0-bar", want: "v6.0.0"},
		{version: "v6.1.0-foo", ref: "v1.0.0-bar", want: "v6.1.0"},
		{version: "v6.1.0-foo", ref: "v4.5.0-bar", want: "v6.1.0"},
		{version: "v6.1.0.1.1.1", ref: "v4.5.0.2.2.2", want: "v6.1.0"},
		{version: "v10.1.0-foo", ref: "v1.0.0", want: "other"},
		{version: "notaversion", ref: "v1.0.0", want: "other"},
		{version: "v6.1.10000000", ref: "v1.0.0-bar", want: "other"},
	}

	for _, c := range cases {
		got := sanitizeVersion(c.version, c.ref)
		if got != c.want {
			t.Errorf("sanitizeVersion(%q, %q) == %q, want %q", c.version, c.ref, got, c.want)
		}
	}
}
