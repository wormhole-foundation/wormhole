# Governor

## Performing a database upgrade

A database upgrade is required whenever the serialized format of a `MessagePublication` changes.
This occurs if a field is added, removed, or changed or if the marshaling format changes.

The Governor has to handle this in a special way because it must be able to read the old format and write the new format.
This is required so that the Governor can continue to monitor transfers and messages that have already been recorded.
It only needs to support the old format for the duration of its sliding window as the old transfers and messages will
automatically dropped after the duration of the window.

### Example upgrade

Commit `1ed88d1` performs a modification of the `MessagePublication` struct and upgrades both the Governor and the Accountant.

### Upgrade Process

When upgrading the database format, follow these steps:

1. **Update prefixes in `node/pkg/db/governor.go`**:
   - Move current prefixes to `old*Prefix` constants (e.g., `transferPrefix` → `oldTransferPrefix`)
   - Increment version number in new prefixes (e.g., `GOV:XFER4:` → `GOV:XFER5:`)
   - Update corresponding length constants

2. **Update database functions**:
   - Ensure `Is*` functions use new prefixes for current format detection
   - Ensure `isOld*` functions use old prefixes for legacy format detection
   - Update `*MsgID` functions to use new prefixes

3. **Update unit tests in `node/pkg/db/governor_test.go`**:
   - Update existing tests to use new prefixes
   - Create separate test functions for each version using naming pattern `TestIs*V[N]`
   - Add function comments explaining version suffix and prefix mapping
   - Example: `TestIsTransferV4` tests `GOV:XFER4:`, `TestIsTransferV3` tests `GOV:XFER3:`

4. **Test coverage**:
   - Ensure both current and legacy formats are tested
   - Include round-trip tests to verify serialization compatibility:
      - Save and load to database
      - Marshal and unmarshal 

#### Database APIs

The actual database CRUD calls should be kept version-agnostic to avoid an explosion in the number of versioned
methods and the accompanying maintenance burden.

### Live Migration

When the Governor restarts and loads its data from the key-value store, it marks which transfers and/or messages
are in the old format and saves them in the new format right away. This minimizes the time needed to support
the two different versions.


