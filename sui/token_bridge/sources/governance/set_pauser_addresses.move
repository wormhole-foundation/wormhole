// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to (re)assign the pauser
/// and unpauser for the Token Bridge emergency pause mechanism.
///
/// The VAA encodes the OWNER address that should receive each capability. The
/// handler MINTS a fresh `PauserCap`/`UnpauserCap`, transfers it to that owner,
/// and records the new cap's object id as the active id in `State` (see
/// `token_bridge::pause`). Because the handler mints and transfers, the active
/// cap is always an owned object — never shared — so only its owner can pause.
///
/// Each `SetPauserAddresses` mints NEW caps. Rotation = new cap to the new
/// owner; any previously minted cap becomes inert (its id no longer matches the
/// recorded active id). A zero/empty owner records `none` (unassigned) and mints
/// nothing.
///
/// On Sui the owner is a 32-byte address (an EOA, or an object that should own
/// the cap). A Sui address is 32 bytes — the same size as on SVM — so the
/// canonical action-4 wire format is unchanged; the Guardian treats the value
/// as opaque length-prefixed bytes and the whitepaper delegates interpretation
/// to the receiving runtime.
///
/// Wire format (action 4, per whitepaper 0003):
/// ```
/// PauserLen(1) | Pauser(PauserLen) | UnpauserLen(1) | Unpauser(UnpauserLen)
/// ```
///
/// Validation:
/// - PauserLen must be 0 (unassigned) or 32 (Sui address size).
/// - UnpauserLen must be 0 (unassigned) or 32.
/// - An all-zero 32-byte value is treated as unassigned (`none`).
/// - No trailing bytes allowed (cursor must be fully consumed).
module token_bridge::set_pauser_addresses {
    use std::option::{Self, Option};
    use sui::object::{ID};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};

    use token_bridge::pause::{Self};
    use token_bridge::state::{Self, State};

    /// Address length is not 0 or 32.
    const E_INVALID_ADDRESS_LENGTH: u64 = 0;

    /// Governance action ID for SetPauserAddresses (canonical, per whitepaper).
    const ACTION_SET_PAUSER_ADDRESSES: u8 = 4;

    /// Expected address size for Sui (32 bytes).
    const SUI_ADDRESS_SIZE: u8 = 32;

    struct GovernanceWitness has drop {}

    /// Event emitted when pauser/unpauser caps are (re)assigned via governance.
    /// `pauser`/`unpauser` are the newly minted cap object ids, or `none` when
    /// the role was left unassigned (no cap minted).
    struct PauserAddressesSet has drop, copy {
        pauser: Option<ID>,
        unpauser: Option<ID>
    }

    /// Create `DecreeTicket` for `SetPauserAddresses` governance VAA.
    /// Uses `authorize_verify_local` (chain-specific, chain == 21 for Sui).
    public fun authorize_governance(
        token_bridge_state: &State
    ): DecreeTicket<GovernanceWitness> {
        governance_message::authorize_verify_local(
            GovernanceWitness {},
            state::governance_chain(token_bridge_state),
            state::governance_contract(token_bridge_state),
            state::governance_module(),
            ACTION_SET_PAUSER_ADDRESSES
        )
    }

    /// Execute the `SetPauserAddresses` governance action. Parses the two owner
    /// addresses, mints a cap for each present owner and transfers it there,
    /// and records the new cap ids (or `none`) as active.
    public fun set_pauser_addresses(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>,
        ctx: &mut TxContext
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let payload =
            governance_message::take_payload(
                state::borrow_mut_consumed_vaas(
                    &latest_only,
                    token_bridge_state
                ),
                receipt
            );

        // Parse the length-prefixed owner addresses (`none` = unassigned).
        let cur = cursor::new(payload);
        let pauser_owner = take_owner_length_prefixed(&mut cur);
        let unpauser_owner = take_owner_length_prefixed(&mut cur);

        // No trailing bytes allowed.
        cursor::destroy_empty(cur);

        // Mint + transfer + record for each role.
        let pauser_id = assign_pauser(token_bridge_state, &latest_only, pauser_owner, ctx);
        let unpauser_id =
            assign_unpauser(token_bridge_state, &latest_only, unpauser_owner, ctx);

        sui::event::emit(
            PauserAddressesSet { pauser: pauser_id, unpauser: unpauser_id }
        );
    }

    /// Mint a `PauserCap` for `owner` and record its id as active. A `none`
    /// owner unassigns the role (records `none`, mints nothing). Returns the
    /// recorded id (`some(cap_id)` or `none`).
    fun assign_pauser(
        token_bridge_state: &mut State,
        latest_only: &state::LatestOnly,
        owner: Option<address>,
        ctx: &mut TxContext
    ): Option<ID> {
        if (option::is_none(&owner)) {
            state::set_pauser(latest_only, token_bridge_state, option::none());
            return option::none()
        };
        let cap = pause::new_pauser_cap(ctx);
        let cap_id = pause::pauser_cap_id(&cap);
        transfer::public_transfer(cap, option::destroy_some(owner));
        state::set_pauser(latest_only, token_bridge_state, option::some(cap_id));
        option::some(cap_id)
    }

    /// Mint an `UnpauserCap` for `owner` and record its id as active. A `none`
    /// owner unassigns the role. Returns the recorded id (`some(cap_id)` or
    /// `none`).
    fun assign_unpauser(
        token_bridge_state: &mut State,
        latest_only: &state::LatestOnly,
        owner: Option<address>,
        ctx: &mut TxContext
    ): Option<ID> {
        if (option::is_none(&owner)) {
            state::set_unpauser(latest_only, token_bridge_state, option::none());
            return option::none()
        };
        let cap = pause::new_unpauser_cap(ctx);
        let cap_id = pause::unpauser_cap_id(&cap);
        transfer::public_transfer(cap, option::destroy_some(owner));
        state::set_unpauser(latest_only, token_bridge_state, option::some(cap_id));
        option::some(cap_id)
    }

    /// Parse a length-prefixed 32-byte owner address from the cursor. Length
    /// must be 0 (returns `none`, unassigned) or SUI_ADDRESS_SIZE (32). A
    /// 32-byte all-zero value is also treated as `none` (unassigned).
    fun take_owner_length_prefixed(cur: &mut cursor::Cursor<u8>): Option<address> {
        let len = bytes::take_u8(cur);
        if (len == 0) {
            return option::none()
        };
        assert!((len as u64) == (SUI_ADDRESS_SIZE as u64), E_INVALID_ADDRESS_LENGTH);

        let addr_bytes = bytes::take_bytes(cur, (SUI_ADDRESS_SIZE as u64));

        // Convert to address. An all-zero address is treated as unassigned.
        let owner = sui::address::from_bytes(addr_bytes);
        if (owner == @0x0) {
            option::none()
        } else {
            option::some(owner)
        }
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_SET_PAUSER_ADDRESSES
    }

    #[test_only]
    /// Parse a raw SetPauserAddresses payload (the part after the governance
    /// header) into the two owner addresses, exercising the exact decode path
    /// used by `set_pauser_addresses` (length validation + no-trailing-bytes).
    public fun parse_payload_test_only(
        payload: vector<u8>
    ): (Option<address>, Option<address>) {
        let cur = cursor::new(payload);
        let pauser_owner = take_owner_length_prefixed(&mut cur);
        let unpauser_owner = take_owner_length_prefixed(&mut cur);
        cursor::destroy_empty(cur);
        (pauser_owner, unpauser_owner)
    }

    #[test_only]
    public fun e_invalid_address_length(): u64 {
        E_INVALID_ADDRESS_LENGTH
    }

    #[test_only]
    /// Directly assign pauser/unpauser owners for tests, bypassing the VAA.
    /// Mints + transfers caps exactly like the governance handler. Returns the
    /// recorded (pauser_id, unpauser_id).
    public fun set_pauser_addresses_test_only(
        token_bridge_state: &mut State,
        pauser_owner: Option<address>,
        unpauser_owner: Option<address>,
        ctx: &mut TxContext
    ): (Option<ID>, Option<ID>) {
        let latest_only = state::assert_latest_only(token_bridge_state);
        let pauser_id =
            assign_pauser(token_bridge_state, &latest_only, pauser_owner, ctx);
        let unpauser_id =
            assign_unpauser(token_bridge_state, &latest_only, unpauser_owner, ctx);
        (pauser_id, unpauser_id)
    }
}
