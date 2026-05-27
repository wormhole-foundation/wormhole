# SUI Token Bridge Pauser — Implementation Plan

## Context

The Wormhole Token Bridge is adding emergency pause support across all chains.
PRs already exist for:

- **Design** — [#4809](https://github.com/wormhole-foundation/wormhole/pull/4809) (whitepaper 0003 update)
- **Guardian** — [#4810](https://github.com/wormhole-foundation/wormhole/pull/4810) (node: VAA generation for `SetPauserAddresses`)
- **EVM** — [#4801](https://github.com/wormhole-foundation/wormhole/pull/4801) (action 4, fixed 20-byte addresses)
- **SVM** — [#4802](https://github.com/wormhole-foundation/wormhole/pull/4802) (action 5, fixed 32-byte pubkeys)

This plan covers the **SUI** implementation.

---

## Design Summary

Two roles — **pauser** and **unpauser** — are configured via a `SetPauserAddresses` governance VAA
(module `"TokenBridge"`, new action ID). When `paused == true`, all user-facing entry points revert.
Governance handlers + `pause()`/`unpause()` remain callable.

### Wire Format (from whitepaper)

```
Module(32)    "TokenBridge" left-padded
Action(1)     new action ID for SUI (see "Action ID" section below)
ChainID(2)    21 (SUI)
Payload       SUI-specific: pauser(32) + unpauser(32)
```

### Action ID Decision

The guardian Go code uses a **single action 4** with length-prefixed addresses on the wire.
However, each runtime interprets a fixed layout:

| Runtime | Action | Address Format |
|---------|--------|----------------|
| EVM     | 4      | Fixed 20 bytes (no length prefix) |
| SVM     | 5      | Fixed 32 bytes (no length prefix) |
| **SUI** | **?**  | Fixed 32 bytes (Sui addresses are 32 bytes) |

**Options:**

1. **Reuse action 5** — SUI addresses are 32 bytes like SVM, same wire layout.
   The SVM contract already uses action 5 with `MODULE = "TokenBridge"`.
   But SUI governance uses `authorize_verify_local()` which checks `chain == 21`,
   so SVM (chain=1) and SUI (chain=21) VAAs cannot collide even with the same action ID.
   **This is safe and mirrors how register_chain (action 1) and upgrade_contract (action 2)
   are shared across all chains with chain-ID differentiation.**

2. **Use action 6** — New SUI-specific action. Cleanest separation but burns an action ID
   for no added security (chain-ID already prevents cross-chain replay).

**Recommendation: Use action 4 (the canonical wire action).** The guardian serializes action 4
with length-prefixed addresses. SUI should parse the length-prefixed format directly, matching
the whitepaper spec. This is what the EVM/SVM PRs *should* be doing too (there's a discrepancy
in those PRs where they parse fixed-layout but the guardian emits length-prefixed). Confirm with
the team which approach wins. If the team prefers per-runtime action IDs, use **action 6**.

**For this plan, we will use action 4 with length-prefixed parsing**, since that matches
the whitepaper and the guardian serialization. If the team decides otherwise, the only change
is the action constant and the deserialization logic.

---

## Architecture

### Approach: Modify Token Bridge State Directly (Not a Companion Package)

Unlike NTT SUI governance (which uses a separate companion package because NTT is parameterized
by `CoinType` and SUI lacks dynamic dispatch), the Token Bridge has a **single shared `State`
object**. We can add pause fields directly to `State` using **dynamic fields** — same pattern
the Token Bridge already uses for version info.

This approach:
- Avoids deploying a separate governance package
- Keeps the pause check inside the existing version-controlled module
- Is delivered as a contract **upgrade** (new version V__0_3_0)
- Uses the existing `DecreeTicket`/`DecreeReceipt` pattern that `register_chain` and
  `upgrade_contract` already use

### State Changes (Dynamic Fields)

We store pause state as dynamic fields on `State.id` to avoid breaking the existing
struct layout:

```move
struct PauserKey has copy, drop, store {}    // → address (32 bytes), 0x0 = unassigned
struct UnpauserKey has copy, drop, store {}  // → address (32 bytes), 0x0 = unassigned
struct PausedKey has copy, drop, store {}    // → bool
```

On first `SetPauserAddresses` governance VAA (or during `migrate`), these fields are
initialized. Before they exist, `is_paused()` returns `false` (backwards compatible).

### Entry Points Requiring `notPaused` Guard

All version-controlled user-facing functions:

| Module | Function | Direction |
|--------|----------|-----------|
| `transfer_tokens` | `transfer_tokens()` | Outbound |
| `transfer_tokens_with_payload` | `transfer_tokens_with_payload()` | Outbound |
| `attest_token` | `attest_token()` | Outbound |
| `complete_transfer` | `authorize_transfer()` | Inbound |
| `complete_transfer_with_payload` | `authorize_transfer()` | Inbound |
| `create_wrapped` | `complete_registration()` | Asset mgmt |
| `create_wrapped` | `update_attestation()` | Asset mgmt |

### Entry Points Exempt from Pause

| Module | Function | Reason |
|--------|----------|--------|
| `register_chain` | `register_chain()` | Governance — must remain callable |
| `upgrade_contract` | `authorize_upgrade()` | Governance — must remain callable |
| `upgrade_contract` | `commit_upgrade()` | Governance — must remain callable |
| `migrate` | `migrate()` | Upgrade — must remain callable |
| `set_pauser_addresses` | (new) | Governance — must remain callable |
| `pause` | (new entry function) | Must be callable to trigger pause |
| `unpause` | (new entry function) | Must be callable when paused |

---

## Implementation Steps

### Phase 0: Tests First (TDD Red)

Write tests before implementation. Test files go in `sui/token_bridge/sources/test/` or alongside
governance modules.

**Test cases for `set_pauser_addresses`:**
1. Success — sets pauser and unpauser via governance VAA
2. Rejects wrong module
3. Rejects wrong chain (not SUI)
4. Rejects replayed VAA (consumed)
5. Can rotate — second VAA overwrites previous addresses
6. Zero address sets role as unassigned

**Test cases for `pause`/`unpause`:**
7. Pauser can pause — `is_paused()` becomes true
8. Non-pauser cannot pause — aborts
9. Unpauser can unpause — `is_paused()` becomes false
10. Non-unpauser cannot unpause — aborts
11. Pause when pauser is unassigned (zero addr) — aborts
12. Unpause when unpauser is unassigned — aborts
13. Pause is idempotent (calling pause when already paused is OK)
14. Unpause is idempotent

**Test cases for `notPaused` guard:**
15. `transfer_tokens` reverts when paused
16. `complete_transfer::authorize_transfer` reverts when paused
17. `attest_token` reverts when paused
18. `create_wrapped::complete_registration` reverts when paused
19. Governance handlers still work when paused (register_chain, upgrade, set_pauser_addresses)
20. Legacy state (no dynamic fields yet) — all operations work as before (unpaused by default)

### Phase 1: State Module Changes (`state.move`)

1. Add dynamic field key structs:
   ```move
   struct PausedKey has copy, drop, store {}
   struct PauserKey has copy, drop, store {}
   struct UnpauserKey has copy, drop, store {}
   ```

2. Add public getter functions:
   ```move
   public fun is_paused(self: &State): bool
   public fun pauser(self: &State): address
   public fun unpauser(self: &State): address
   ```
   - `is_paused()` returns `false` if `PausedKey` dynamic field doesn't exist (backwards compat)
   - `pauser()`/`unpauser()` return `@0x0` if field doesn't exist

3. Add friend-only mutation functions:
   ```move
   public(friend) fun set_paused(_: &LatestOnly, self: &mut State, paused: bool)
   public(friend) fun set_pauser_address(_: &LatestOnly, self: &mut State, pauser: address)
   public(friend) fun set_unpauser_address(_: &LatestOnly, self: &mut State, unpauser: address)
   ```

4. Add `assert_not_paused` helper:
   ```move
   public(friend) fun assert_not_paused(self: &State)
   ```
   Aborts with `E_PAUSED` if `is_paused()` returns true.

5. Add new friends:
   ```move
   friend token_bridge::set_pauser_addresses;
   friend token_bridge::pause;
   ```

### Phase 2: Version Control (`version_control.move`)

1. Add new version `V__0_3_0`:
   ```move
   struct V__0_3_0 has store, drop, copy {}
   ```

2. Update `current_version()` to return `V__0_3_0`.
3. Set `previous_version()` to `V__0_2_0`.

### Phase 3: Governance — `set_pauser_addresses.move` (New File)

New module: `token_bridge::set_pauser_addresses`

```
sui/token_bridge/sources/governance/set_pauser_addresses.move
```

**Constants:**
```move
const ACTION_SET_PAUSER_ADDRESSES: u8 = 4;
```

**Structs:**
```move
struct GovernanceWitness has drop {}
```

**Functions:**

```move
/// Create DecreeTicket for SetPauserAddresses governance VAA.
/// Uses authorize_verify_local (chain-specific, chain == 21).
public fun authorize_governance(
    token_bridge_state: &State
): DecreeTicket<GovernanceWitness>

/// Execute the SetPauserAddresses governance action.
/// Consumes the DecreeReceipt, parses the payload, and updates state.
public fun set_pauser_addresses(
    token_bridge_state: &mut State,
    receipt: DecreeReceipt<GovernanceWitness>
)
```

**Payload Parsing (action 4, length-prefixed per whitepaper):**
```
PauserLen(1)   | Pauser(PauserLen)   | UnpauserLen(1) | Unpauser(UnpauserLen)
```

Validation:
- If PauserLen > 0: must be exactly 32 bytes (SUI address size), else abort
- If PauserLen == 0: set pauser to `@0x0` (unassigned)
- Same for UnpauserLen/Unpauser
- All-zero 32-byte address treated same as unassigned (set to `@0x0`)
- No trailing bytes allowed (cursor must be fully consumed)

### Phase 4: Pause/Unpause Entry Functions (New File)

New module: `token_bridge::pause`

```
sui/token_bridge/sources/pause.move
```

**Entry functions:**

```move
/// Pause the token bridge. Only callable by the configured pauser.
/// Aborts if pauser is unassigned (@0x0).
public entry fun pause(
    token_bridge_state: &mut State,
    ctx: &TxContext
)

/// Unpause the token bridge. Only callable by the configured unpauser.
/// Aborts if unpauser is unassigned (@0x0).
/// NOT guarded by assert_not_paused (must be callable when paused).
public entry fun unpause(
    token_bridge_state: &mut State,
    ctx: &TxContext
)
```

**Logic:**
1. `assert_latest_only(state)` — version check
2. Read configured pauser/unpauser from state
3. Assert configured address != `@0x0` (`E_PAUSER_NOT_CONFIGURED`)
4. Assert `tx_context::sender(ctx) == configured_address` (`E_NOT_PAUSER` / `E_NOT_UNPAUSER`)
5. Call `state::set_paused(latest_only, state, true/false)`

### Phase 5: Add Pause Guards to Existing Modules

In each guarded module, add `state::assert_not_paused(&state)` right after
`state::assert_latest_only(&state)`. This ensures the pause check happens at the
same point as the version check — early, before any state mutation.

Files to modify:
1. `transfer_tokens.move` — in `transfer_tokens()`
2. `transfer_tokens_with_payload.move` — in `transfer_tokens_with_payload()`
3. `attest_token.move` — in `attest_token()`
4. `complete_transfer.move` — in `authorize_transfer()`
5. `complete_transfer_with_payload.move` — in `authorize_transfer()`
6. `create_wrapped.move` — in `complete_registration()` and `update_attestation()`

Each change is a single line addition:
```move
let latest_only = state::assert_latest_only(&token_bridge_state);
state::assert_not_paused(token_bridge_state);  // <-- NEW
```

### Phase 6: Migration (`migrate.move`)

Update `migrate__v__0_3_0()` to initialize dynamic fields:

```move
public(friend) fun migrate__v__0_3_0(state: &mut State) {
    // Initialize pause dynamic fields with defaults.
    // paused = false, pauser = @0x0, unpauser = @0x0
    state::init_pause_state(state);
}
```

This ensures the dynamic fields exist after upgrade, even before any governance
VAA is submitted. The `is_paused()` getter already handles the case where fields
don't exist (returns false), but initializing them during migration is cleaner.

### Phase 7: Guardian Node Changes (if needed)

The guardian Go code already serializes action 4 with length-prefixed addresses.
If we use action 4 on SUI, no guardian changes are needed — the same VAA
targeting chain 21 (SUI) will work.

If the team decides to use a SUI-specific action ID (e.g., 6), we need to:
1. Add `ActionTokenBridgeSetPauserAddressesSui GovernanceAction = 6` in `sdk/vaa/payloads.go`
2. Add a SUI serialization path in `adminserver.go`

### Phase 8: Run Tests (TDD Green)

1. `cd sui/token_bridge && sui move test`
2. All Phase 0 tests should pass
3. Existing tests must continue passing (backwards compatibility)

### Phase 9: Code Review

Run code-reviewer agent on all changed files.

---

## File Change Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `sui/token_bridge/sources/state.move` | Modify | Add pause dynamic fields, getters, setters, `assert_not_paused` |
| `sui/token_bridge/sources/version_control.move` | Modify | Add V__0_3_0, update current/previous |
| `sui/token_bridge/sources/governance/set_pauser_addresses.move` | **New** | Governance action handler |
| `sui/token_bridge/sources/pause.move` | **New** | `pause()`/`unpause()` entry functions |
| `sui/token_bridge/sources/migrate.move` | Modify | Add `migrate__v__0_3_0()` |
| `sui/token_bridge/sources/transfer_tokens.move` | Modify | Add `assert_not_paused` (1 line) |
| `sui/token_bridge/sources/transfer_tokens_with_payload.move` | Modify | Add `assert_not_paused` (1 line) |
| `sui/token_bridge/sources/attest_token.move` | Modify | Add `assert_not_paused` (1 line) |
| `sui/token_bridge/sources/complete_transfer.move` | Modify | Add `assert_not_paused` (1 line) |
| `sui/token_bridge/sources/complete_transfer_with_payload.move` | Modify | Add `assert_not_paused` (1 line) |
| `sui/token_bridge/sources/create_wrapped.move` | Modify | Add `assert_not_paused` (2 lines) |

---

## Open Questions for Team

1. **Action ID**: Use action 4 (whitepaper canonical, length-prefixed) or a SUI-specific
   action ID (5 or 6)? This plan assumes action 4. The EVM/SVM PRs have a discrepancy
   where the guardian emits length-prefixed action 4 but the contracts parse fixed-layout
   with action 4 (EVM) / action 5 (SVM). Need team alignment on the intended approach.

2. **Events**: Should we emit Move events for `Paused`, `Unpaused`, `PauserAddressesSet`?
   SUI supports events via `sui::event::emit`. The EVM PR emits events; the SVM PR doesn't
   (Solana doesn't have events in the same sense). Recommendation: yes, emit events.

3. **Dynamic fields vs struct fields**: This plan uses dynamic fields to avoid breaking
   the existing `State` struct. Alternative: modify the struct directly in the migration.
   Dynamic fields are safer for upgrades (no storage layout assumptions).

4. **Backwards compatibility**: The `is_paused()` getter handles missing dynamic fields
   gracefully (returns false). Do we also want to handle the case where `set_pauser_addresses`
   is called before migration? This plan says no — the governance module requires
   `LatestOnly` which requires V__0_3_0, so migration must happen first.
