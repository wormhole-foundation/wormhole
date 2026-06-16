package channelcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// runFixture runs the Analyzer against testdata/src/<pkg> under a transient
// settings override. It restores the previous settings on return so tests do
// not interfere with each other.
func runFixture(t *testing.T, pkg string, override Settings) {
	t.Helper()

	saved := settings
	defer func() { settings = saved }()

	// Default matches the flag defaults (blocking on, everything else off).
	settings = Settings{CheckBlockingSends: true}

	settings.CheckUnbufferedChannels = override.CheckUnbufferedChannels
	settings.CheckBlockingSends = override.CheckBlockingSends
	settings.CheckEmptyDefault = override.CheckEmptyDefault
	settings.CheckBufferAmount = override.CheckBufferAmount
	settings.IgnoreChannelsByName = override.IgnoreChannelsByName
	settings.ignoreChannelNames = buildIgnoreSet(override.IgnoreChannelsByName)

	analysistest.Run(t, analysistest.TestData(), Analyzer, pkg)
}

func defaultSettings() Settings {
	return Settings{CheckBlockingSends: true}
}

func TestBlockingSend(t *testing.T) {
	runFixture(t, "blocking_send", defaultSettings())
}

func TestEmptyDefault(t *testing.T) {
	runFixture(t, "empty_default", Settings{CheckBlockingSends: true, CheckEmptyDefault: true})
}

func TestCtxDoneOnly(t *testing.T) {
	runFixture(t, "ctx_done_only", defaultSettings())
}

func TestTimerSafe(t *testing.T) {
	runFixture(t, "timer_safe", defaultSettings())
}

func TestDefaultSafe(t *testing.T) {
	runFixture(t, "default_safe", defaultSettings())
}

func TestUnbufferedChannel(t *testing.T) {
	runFixture(t, "unbuffered_chan", Settings{
		CheckUnbufferedChannels: true,
		CheckBlockingSends:      false,
	})
}

func TestBufferTooLarge(t *testing.T) {
	runFixture(t, "buffer_too_large", Settings{
		CheckBlockingSends: false,
		CheckBufferAmount:  10,
	})
}

func TestBlockingDisabled(t *testing.T) {
	runFixture(t, "blocking_disabled", Settings{CheckBlockingSends: false})
}

func TestBlockingDisabledSuppressesEmptyDefault(t *testing.T) {
	runFixture(t, "blocking_disabled_empty_default", Settings{CheckBlockingSends: false})
}

func TestBlockingDisabledSuppressesCtxDone(t *testing.T) {
	runFixture(t, "blocking_disabled_ctx_done", Settings{CheckBlockingSends: false})
}

func TestUnbufferedDisabledByDefault(t *testing.T) {
	// Default config has CheckUnbufferedChannels=false. unbuffered_chan.go's
	// want marker only fires when the check is on; with the check off we need
	// a fixture without a // want marker.
	runFixture(t, "unbuffered_disabled", defaultSettings())
}

func TestEscapeOtherAlone(t *testing.T) {
	runFixture(t, "escape_other_alone", defaultSettings())
}

func TestDerivedCtxDone(t *testing.T) {
	runFixture(t, "derived_ctx_done", defaultSettings())
}

func TestFakeDoneCounter(t *testing.T) {
	runFixture(t, "fake_done_counter", defaultSettings())
}

func TestSendInCaseBody(t *testing.T) {
	runFixture(t, "send_in_case_body", defaultSettings())
}

func TestIgnoreChannelsByName(t *testing.T) {
	runFixture(t, "ignore_by_name", Settings{
		CheckBlockingSends:   true,
		IgnoreChannelsByName: []string{"ignoreMe"},
	})
}

func TestIgnoreSuppressesEmptyDefault(t *testing.T) {
	runFixture(t, "ignore_empty_default", Settings{
		CheckBlockingSends:   true,
		IgnoreChannelsByName: []string{"ignoreMe"},
	})
}

func TestIgnoreSuppressesCtxDone(t *testing.T) {
	runFixture(t, "ignore_ctx_done", Settings{
		CheckBlockingSends:   true,
		IgnoreChannelsByName: []string{"ignoreMe"},
	})
}

func TestIgnorePartialStillFires(t *testing.T) {
	runFixture(t, "ignore_partial", Settings{
		CheckBlockingSends:   true,
		IgnoreChannelsByName: []string{"ignoreMe"},
	})
}

func TestBufferMaxDisabledByDefault(t *testing.T) {
	// Default has CheckBufferAmount=0 which disables the check. The
	// buffer_too_large fixture's // want marker requires the check on, so a
	// no-want fixture is used here.
	runFixture(t, "buffer_disabled", Settings{CheckBlockingSends: false})
}
