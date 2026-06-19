package main

import "testing"

func TestShouldExclude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pkg  string
		want bool
	}{
		{
			name: "testutils package",
			pkg:  "github.com/certusone/wormhole/node/pkg/testutils",
			want: true,
		},
		{
			name: "testutils child package",
			pkg:  "github.com/certusone/wormhole/node/pkg/testutils/builders",
			want: true,
		},
		{
			name: "normal package",
			pkg:  "github.com/certusone/wormhole/node/pkg/processor",
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldExclude(tt.pkg); got != tt.want {
				t.Fatalf("shouldExclude(%q) = %t, want %t", tt.pkg, got, tt.want)
			}
		})
	}
}
