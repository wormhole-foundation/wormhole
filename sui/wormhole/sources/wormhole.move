module wormhole::wormhole {
    use sui::sui::{SUI};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};

    // use wormhole::myu16 as u16;
    // use wormhole::myu32::{Self as u32, U32};
    // use wormhole::external_address::{Self};

    const E_INSUFFICIENT_FEE: u64 = 0;

// -----------------------------------------------------------------------------
// Sending messages
    public entry fun publish_message(
        state: &State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
        ctx: &mut TxContext,
    ) {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        assert!(expected_fee <= coin::value(&message_fee), E_INSUFFICIENT_FEE);
        // deposit the fees into the wormhole account
        transfer::transfer(message_fee, @wormhole);
        let sequence = 0;
        state::publish_event(
            sequence,
            nonce,
            payload,
            ctx,
        );
        //sequence
    }

// -----------------------------------------------------------------------------
// Contract initialization

    ///// Initializes State with correct args and subsequently makes
    ///// State a shared object.
    // public entry fun init_wormhole(
    //     //state: &mut State,
    //     chain_id: u64,
    //     governance_chain_id: u64,
    //     governance_contract: vector<u8>,
    //     initial_guardian: vector<u8>
    // ) {
    //     let message_fee = 0;
    //     init_internal(
    //         //state,
    //         chain_id,
    //         governance_chain_id,
    //         governance_contract,
    //         initial_guardian,
    //         u32::from_u64(86400),
    //         message_fee
    //     )
    // }

    // fun init_internal(
    //     //state: &mut State,
    //     chain_id: u64,
    //     governance_chain_id: u64,
    //     governance_contract: vector<u8>,
    //     initial_guardian: vector<u8>,
    //     guardian_set_expiry: U32,
    //     message_fee: u64,
    // ) {
    //     state::init_wormhole_state(
    //         //&wormhole,
    //         u16::from_u64(chain_id),
    //         u16::from_u64(governance_chain_id),
    //         external_address::from_bytes(governance_contract),
    //         guardian_set_expiry,
    //         message_fee,
    //         signer_cap
    //     );
    //     state::store_guardian_set(
    //         create_guardian_set(
    //             u32::from_u64(0),
    //             vector[create_guardian(initial_guardian)]
    //         )
    //     );
    // }
}

//     #[test_only]
//     /// Initialise a dummy contract for testing. Returns the wormhole signer.
//     public fun init_test(
//         chain_id: u64,
//         governance_chain_id: u64,
//         governance_contract: vector<u8>,
//         initial_guardian: vector<u8>,
//         message_fee: u64,
//     ): signer {
//         let deployer = account::create_account_for_test(@0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b);
//         let (wormhole, signer_cap) = account::create_resource_account(&deployer, b"wormhole");
//         init_internal(
//             signer_cap,
//             chain_id,
//             governance_chain_id,
//             governance_contract,
//             initial_guardian,
//             u32::from_u64(86400),
//             message_fee
//         );
//         wormhole
//     }
// }

// #[test_only]
// module wormhole::wormhole_test {
//     use 0x1::hash;
//     use wormhole::wormhole;
//     use wormhole::keccak256::keccak256;
//     use aptos_framework::aptos_coin::{Self};
//     use aptos_framework::coin;

//     // public so we an re-use this in token_bridge test
//     public fun setup(message_fee: u64) {
//         let aptos_framework = std::account::create_account_for_test(@aptos_framework);
//         std::timestamp::set_time_has_started_for_testing(&aptos_framework);
//         wormhole::init_test(
//             22,
//             1,
//             x"0000000000000000000000000000000000000000000000000000000000000004",
//             x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
//             message_fee
//         );
//     }

//     #[test]
//     public fun test_hash() {
//         assert!(hash::sha3_256(vector[0]) == x"5d53469f20fef4f8eab52b88044ede69c77a6a68a60728609fc4a65ff531e7d0", 0);
//         assert!(keccak256(vector[0]) == x"bc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a", 0);
//     }

//     #[test(aptos_framework = @aptos_framework)]
//     public fun test_publish_message(aptos_framework: &signer) {
//         setup(100);

//         let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
//         let fees = coin::mint(100, &mint_cap);

//         let emitter_cap = wormhole::register_emitter();

//         wormhole::publish_message(
//             &mut emitter_cap,
//             0,
//             b"hi mom",
//             fees
//         );

//         //TODO - check if event is actually emitted?

//         wormhole::emitter::destroy_emitter_cap(emitter_cap);
//         coin::destroy_mint_cap(mint_cap);
//         coin::destroy_burn_cap(burn_cap);
//     }

//     #[test]
//     #[expected_failure(abort_code = 0x0)]
//     public fun test_publish_message_insufficient_fee() {
//         setup(100);
//         let emitter_cap = wormhole::register_emitter();

//         wormhole::publish_message(
//             &mut emitter_cap,
//             0,
//             b"hi mom",
//             coin::zero()
//         );
//         wormhole::emitter::destroy_emitter_cap(emitter_cap);
//     }
// }