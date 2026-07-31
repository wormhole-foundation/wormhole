package channelcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func defaultSettings() Settings {
	return Settings{CheckBlockingSends: true}
}

// TestChannelCheck runs the Analyzer against each testdata/src/<fixture>
// package under the case's settings. The analyzer reads package-level
// settings, so the subtests mutate global state and must not run in parallel.
func TestChannelCheck(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		settings Settings
	}{
		{
			name:     "blocking send",
			fixture:  "blocking_send",
			settings: defaultSettings(),
		},
		{
			name:     "empty default",
			fixture:  "empty_default",
			settings: Settings{CheckBlockingSends: true, CheckEmptyDefault: true},
		},
		{
			name:     "ctx done only",
			fixture:  "ctx_done_only",
			settings: defaultSettings(),
		},
		{
			name:     "timer safe",
			fixture:  "timer_safe",
			settings: defaultSettings(),
		},
		{
			name:     "default safe",
			fixture:  "default_safe",
			settings: defaultSettings(),
		},
		{
			name:    "unbuffered channel",
			fixture: "unbuffered_chan",
			settings: Settings{
				CheckUnbufferedChannels: true,
				CheckBlockingSends:      false,
			},
		},
		{
			name:    "buffer too large",
			fixture: "buffer_too_large",
			settings: Settings{
				CheckBlockingSends: false,
				CheckBufferAmount:  10,
			},
		},
		{
			name:     "blocking disabled",
			fixture:  "blocking_disabled",
			settings: Settings{CheckBlockingSends: false},
		},
		{
			name:     "blocking disabled suppresses empty default",
			fixture:  "blocking_disabled_empty_default",
			settings: Settings{CheckBlockingSends: false},
		},
		{
			name:     "blocking disabled suppresses ctx done",
			fixture:  "blocking_disabled_ctx_done",
			settings: Settings{CheckBlockingSends: false},
		},
		{
			// Default config has CheckUnbufferedChannels=false. unbuffered_chan.go's
			// want marker only fires when the check is on; with the check off we need
			// a fixture without a // want marker.
			name:     "unbuffered disabled by default",
			fixture:  "unbuffered_disabled",
			settings: defaultSettings(),
		},
		{
			name:     "escape other alone",
			fixture:  "escape_other_alone",
			settings: defaultSettings(),
		},
		{
			name:     "derived ctx done",
			fixture:  "derived_ctx_done",
			settings: defaultSettings(),
		},
		{
			name:     "fake done counter",
			fixture:  "fake_done_counter",
			settings: defaultSettings(),
		},
		{
			name:     "send in case body",
			fixture:  "send_in_case_body",
			settings: defaultSettings(),
		},
		{
			name:    "ignore channels by name",
			fixture: "ignore_by_name",
			settings: Settings{
				CheckBlockingSends:   true,
				IgnoreChannelsByName: []string{"ignoreMe"},
			},
		},
		{
			name:    "ignore suppresses empty default",
			fixture: "ignore_empty_default",
			settings: Settings{
				CheckBlockingSends:   true,
				IgnoreChannelsByName: []string{"ignoreMe"},
			},
		},
		{
			name:    "ignore suppresses ctx done",
			fixture: "ignore_ctx_done",
			settings: Settings{
				CheckBlockingSends:   true,
				IgnoreChannelsByName: []string{"ignoreMe"},
			},
		},
		{
			name:    "ignore partial still fires",
			fixture: "ignore_partial",
			settings: Settings{
				CheckBlockingSends:   true,
				IgnoreChannelsByName: []string{"ignoreMe"},
			},
		},
		{
			// Default has CheckBufferAmount=0 which disables the check. The
			// buffer_too_large fixture's // want marker requires the check on, so a
			// no-want fixture is used here.
			name:     "buffer max disabled by default",
			fixture:  "buffer_disabled",
			settings: Settings{CheckBlockingSends: false},
		},
	}

	saved := settings
	defer func() { settings = saved }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings = tt.settings
			settings.ignoreChannelNames = buildIgnoreSet(tt.settings.IgnoreChannelsByName)

			analysistest.Run(t, analysistest.TestData(), Analyzer, tt.fixture)
		})
	}
}
