package apptesting

import sdk "github.com/cosmos/cosmos-sdk/types"

// AssertEventEmitted asserts that ctx's event manager has emitted the given number of events
// of the given type.
func (s *KeeperTestHelper) AssertEventEmitted(ctx sdk.Context, eventTypeExpected string, numEventsExpected int) {
	allEvents := ctx.EventManager().Events()
	// filter out other events
	actualEvents := make([]sdk.Event, 0)
	for _, event := range allEvents {
		if event.Type == eventTypeExpected {
			actualEvents = append(actualEvents, event)
		}
	}
	s.Equal(numEventsExpected, len(actualEvents))
}
