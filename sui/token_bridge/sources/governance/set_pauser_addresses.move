// SPDX-License-Identifier: Apache 2

/// This module implements handling a governance VAA to set the pauser and
/// unpauser addresses on the Token Bridge. These addresses control the
/// emergency pause mechanism.
///
/// Wire format (action 4, per whitepaper 0003):
/// ```
/// PauserLen(1) | Pauser(PauserLen) | UnpauserLen(1) | Unpauser(UnpauserLen)
/// ```
///
/// Validation:
/// - PauserLen must be 0 (unassigned) or 32 (Sui address size).
/// - UnpauserLen must be 0 (unassigned) or 32 (Sui address size).
/// - An all-zero 32-byte address is treated as unassigned (@0x0).
/// - No trailing bytes allowed (cursor must be fully consumed).
module token_bridge::set_pauser_addresses {
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::governance_message::{Self, DecreeTicket, DecreeReceipt};

    use token_bridge::state::{Self, State};

    /// Address length is not 0 or 32.
    const E_INVALID_ADDRESS_LENGTH: u64 = 0;

    /// Governance action ID for SetPauserAddresses (canonical, per whitepaper).
    const ACTION_SET_PAUSER_ADDRESSES: u8 = 4;

    /// Expected address size for Sui (32 bytes).
    const SUI_ADDRESS_SIZE: u8 = 32;

    struct GovernanceWitness has drop {}

    /// Event emitted when pauser addresses are updated via governance.
    struct PauserAddressesSet has drop, copy {
        pauser: address,
        unpauser: address
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

    /// Execute the `SetPauserAddresses` governance action.
    /// Consumes the `DecreeReceipt`, parses the length-prefixed payload,
    /// and updates state.
    public fun set_pauser_addresses(
        token_bridge_state: &mut State,
        receipt: DecreeReceipt<GovernanceWitness>
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

        // Parse the length-prefixed payload.
        let cur = cursor::new(payload);

        let pauser = take_address_length_prefixed(&mut cur);
        let unpauser = take_address_length_prefixed(&mut cur);

        // No trailing bytes allowed.
        cursor::destroy_empty(cur);

        // Update state.
        state::set_pauser_address(&latest_only, token_bridge_state, pauser);
        state::set_unpauser_address(
            &latest_only,
            token_bridge_state,
            unpauser
        );

        // Emit event.
        sui::event::emit(PauserAddressesSet { pauser, unpauser });
    }

    /// Parse a length-prefixed address from the cursor.
    /// Length must be 0 (returns @0x0) or SUI_ADDRESS_SIZE (32).
    /// An all-zero 32-byte address is also treated as @0x0.
    fun take_address_length_prefixed(cur: &mut cursor::Cursor<u8>): address {
        let len = bytes::take_u8(cur);
        if (len == 0) {
            return @0x0
        };
        assert!((len as u64) == (SUI_ADDRESS_SIZE as u64), E_INVALID_ADDRESS_LENGTH);

        let addr_bytes = bytes::take_bytes(cur, (SUI_ADDRESS_SIZE as u64));

        // Convert to address.
        sui::address::from_bytes(addr_bytes)
    }

    #[test_only]
    public fun action(): u8 {
        ACTION_SET_PAUSER_ADDRESSES
    }

    #[test_only]
    /// Directly set pauser addresses for tests, bypassing governance VAA.
    public fun set_pauser_addresses_test_only(
        token_bridge_state: &mut State,
        pauser: address,
        unpauser: address
    ) {
        let latest_only = state::assert_latest_only(token_bridge_state);
        state::set_pauser_address(&latest_only, token_bridge_state, pauser);
        state::set_unpauser_address(
            &latest_only,
            token_bridge_state,
            unpauser
        );
    }
}
