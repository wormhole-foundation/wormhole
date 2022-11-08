module token_bridge::bridge_state {
   use std::vector::{Self};
   use std::option::{Self, Option};

   use sui::object::{Self, UID};
   use sui::vec_map::{Self, VecMap};
   use sui::dynamic_object_field::{Self};
   use sui::tx_context::{TxContext};
   use sui::coin::{Coin};
   use sui::transfer::{Self};
   use sui::tx_context::{Self};
   use sui::sui::SUI;
   use sui::object_table::{Self};

   use token_bridge::treasury::{Self, CoinStore, TreasuryCapStore};

   use wormhole::external_address::{Self, ExternalAddress};
   use wormhole::myu16::{Self as u16, U16};
   use wormhole::wormhole::{Self};
   use wormhole::state::{State};
   use wormhole::emitter::{EmitterCapability};

   const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
   const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
   const E_WRAPPING_NATIVE_COIN: u64 = 2;
   const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

   friend token_bridge::vaa;
   friend token_bridge::register_chain;
   friend token_bridge::wrapped;
   friend token_bridge::complete_transfer;
   friend token_bridge::transfer_tokens;
   #[test_only]
   friend token_bridge::test_bridge_state;

   /// TODO - The origin chain and address of a token.  In case of native tokens
   ///        what do we set the token_address to? For Aptos it was the hash of the
   ///        deployer + module_name + struct_name, for Sui might have to do differently
   struct OriginInfo has store, copy, drop {
      token_chain: U16,
      token_address: ExternalAddress,
   }

   public fun create_origin_info(token_chain: U16, token_address: ExternalAddress): OriginInfo {
      return OriginInfo {
         token_chain,
         token_address
      }
   }

   struct Unit has key, store {id: UID,} // for turning object_table into a set

   // TODO - move to newtypes
   struct RegisteredEmitter {
      emitter: ExternalAddress
   }

   // Treasury caps, token stores, consumed VAAs, registered emitters, etc.
   // are stored as dynamic fields of bridge state.
   struct BridgeState has key, store {
      id: UID,
      governance_chain_id: U16,
      governance_contract: ExternalAddress,

      /// Set of consumed VAA hashes
      consumed_vaas: object_table::ObjectTable<vector<u8>, Unit>,

      /// Token bridge owned emitter capability
      emitter_cap: option::Option<EmitterCapability>,

      // Mapping of bridge contracts on other chains
      // TODO - figure out if it is OK to keep this?
      //        there will likely never be a few 100s of other bridge contracts
      registered_emitters: VecMap<U16, ExternalAddress>,
   }

   fun init(ctx: &mut TxContext) {
        transfer::transfer(BridgeState {
            id: object::new(ctx),
            governance_chain_id: u16::from_u64(0),
            governance_contract: external_address::from_bytes(vector::empty<u8>()),
            consumed_vaas: object_table::new<vector<u8>, Unit>(ctx),
            emitter_cap: option::none<EmitterCapability>(),
            registered_emitters: vec_map::empty<U16, ExternalAddress>(),
        }, tx_context::sender(ctx));
    }

   #[test_only]
   public fun test_init(ctx: &mut TxContext) {
      transfer::transfer(BridgeState {
            id: object::new(ctx),
            governance_chain_id: u16::from_u64(0),
            governance_contract: external_address::from_bytes(vector::empty<u8>()),
            consumed_vaas: object_table::new<vector<u8>, Unit>(ctx),
            emitter_cap: option::none<EmitterCapability>(),
            registered_emitters: vec_map::empty<U16, ExternalAddress>(),
        }, tx_context::sender(ctx));
   }

   // converts owned state object into a shared object, so that anyone can get a reference to &mut State
   // and pass it into various functions
   public entry fun init_and_share_state(
      state: BridgeState,
      emitter_cap: EmitterCapability,
      governance_chain_id: u64,
      governance_contract: vector<u8>,
      _ctx: &mut TxContext
   ) {
      option::fill<EmitterCapability>(&mut state.emitter_cap, emitter_cap);
      set_governance_chain_id(&mut state, u16::from_u64(governance_chain_id));
      set_governance_contract(&mut state, external_address::from_bytes(governance_contract));

      // permanently shares state
      transfer::share_object(state);
   }

   public(friend) fun deposit<CoinType>(
      bridge_state: &mut BridgeState,
      coin: Coin<CoinType>,
      origin_info: OriginInfo,
      _ctx: &mut TxContext
   ) {

      // TODO: confirm that CoinStore<CoinType> exists as a child object of bridge_state
      //       if it is not a child object, initialize a CoinStore and transfer it to bridge
      //       if it is, obtain a reference to it

      let coin_store = dynamic_object_field::borrow_mut<OriginInfo, CoinStore<CoinType>>(&mut bridge_state.id, origin_info);
      treasury::deposit<CoinType>(coin_store, coin);
   }

   public(friend) fun withdraw<CoinType>(
      bridge_state: &mut BridgeState,
      value: u64,
      origin_info: OriginInfo,
      ctx: &mut TxContext
   ): Coin<CoinType> {

      // TODO: confirm that CoinStore<CoinType> exists as a child object of bridge_state
      //      if it is not a child object, initialize a CoinStore and transfer it to bridge
      //      if it is, obtain a reference to it

      let coin_store = dynamic_object_field::borrow_mut<OriginInfo, CoinStore<CoinType>>(&mut bridge_state.id, origin_info);
      let coins = treasury::withdraw<CoinType>(coin_store, value, ctx);
      return coins
   }

   public(friend) fun mint<CoinType>(
      state: &mut BridgeState,
      value: u64,
      origin_info: OriginInfo,
      ctx: &mut TxContext,
   ): Coin<CoinType> {
      let treasury_cap_store = dynamic_object_field::borrow_mut<OriginInfo, TreasuryCapStore<CoinType>>(&mut state.id, origin_info);
      let coins = treasury::mint<CoinType>(treasury_cap_store, value, ctx);
      return coins
   }

   public(friend) fun burn<CoinType>(
      state: &mut BridgeState,
      coin: Coin<CoinType>,
      origin_info: OriginInfo,
   ) {
      let treasury_cap_store = dynamic_object_field::borrow_mut<OriginInfo, TreasuryCapStore<CoinType>>(&mut state.id, origin_info);
      treasury::burn<CoinType>(treasury_cap_store, coin);
   }

   public(friend) fun publish_message(
      wormhole_state: &mut State,
      bridge_state: &mut BridgeState,
      nonce: u64,
      payload: vector<u8>,
      message_fee: Coin<SUI>,
      ctx: &mut TxContext
   ) {
      wormhole::publish_message(
         option::borrow_mut<EmitterCapability>(&mut bridge_state.emitter_cap),
         wormhole_state,
         nonce,
         payload,
         message_fee,
         ctx
      )
   }

   // getters

   public fun vaa_is_consumed(state: &BridgeState, hash: vector<u8>): bool {
      object_table::contains<vector<u8>, Unit>(&state.consumed_vaas, hash)//create_consumed_vaa(hash))
   }

   public fun get_governance_chain_id(state: &BridgeState): U16 {
      state.governance_chain_id
   }

   public fun get_governance_contract(state: &BridgeState): ExternalAddress {
      state.governance_contract
   }

   public fun get_registered_emitter(state: &BridgeState, chain_id: &U16): Option<ExternalAddress> {
      if (vec_map::contains(&state.registered_emitters, chain_id)) {
         option::some(*vec_map::get(&state.registered_emitters, chain_id))
      } else {
         option::none()
      }
   }

   // setters

   public(friend) fun set_governance_chain_id(state: &mut BridgeState, governance_chain_id: U16) {
      state.governance_chain_id = governance_chain_id;
   }

   #[test_only]
   public fun test_set_governance_chain_id(state: &mut BridgeState, governance_chain_id: U16) {
      state.governance_chain_id = governance_chain_id;
   }

   public(friend) fun set_governance_contract(state: &mut BridgeState, governance_contract: ExternalAddress) {
      state.governance_contract = governance_contract;
   }

   public(friend) fun set_registered_emitter(state: &mut BridgeState, chain_id: U16, emitter: ExternalAddress) {
      vec_map::insert<U16, ExternalAddress>(&mut state.registered_emitters, chain_id, emitter);
   }

   // dynamic ops

   // store the treasury_cap_store as a dynamic field of bridge state
   public(friend) fun store_treasury_cap<T>(state: &mut BridgeState, origin_info: OriginInfo, treasury_cap_store: treasury::TreasuryCapStore<T>) {
      dynamic_object_field::add<OriginInfo, treasury::TreasuryCapStore<T>>(&mut state.id, origin_info, treasury_cap_store);
   }

   // store the coin store as a dynamic field of bridge state
   public(friend) fun store_coin_store<T>(state: &mut BridgeState, origin_info: OriginInfo, treasury_coin_store: treasury::CoinStore<T>) {
      dynamic_object_field::add<OriginInfo, treasury::CoinStore<T>>(&mut state.id, origin_info, treasury_coin_store);
   }

   public(friend) fun store_consumed_vaa(state: &mut BridgeState, vaa: vector<u8>, ctx: &mut TxContext) {
      object_table::add<vector<u8>, Unit>(&mut state.consumed_vaas, vaa, Unit{id: object::new(ctx)});
   }

}

#[test_only]
module token_bridge::test_bridge_state{
   use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_from_address, return_to_address, take_shared, return_shared};
   //use std::vector::{Self};

   use wormhole::state::{Self as wormhole_state, State};
   use wormhole::myu16::{Self as u16};
   use wormhole::wormhole::{Self};

   use token_bridge::bridge_state::{Self, BridgeState, init_and_share_state};

   fun scenario(): Scenario { test_scenario::begin(@0x123233) }
   fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

   #[test]
   fun test_state_setters() {
      test_state_setters_(scenario())
   }

   fun test_state_setters_(test: Scenario) {
         let (admin, _, _) = people();

         // init wormhole state
         next_tx(&mut test, admin); {
            wormhole_state::test_init(ctx(&mut test));
         };

         // init token bridge state
         next_tx(&mut test, admin); {
            bridge_state::test_init(ctx(&mut test));
         };

         // register for an emitter cap for token bridge, then init and share
         // token bridge state
         next_tx(&mut test, admin); {
            let wormhole_state = take_from_address<State>(&test, admin);
            let bridge_state = take_from_address<BridgeState>(&test, admin);
            let my_emitter = wormhole::register_emitter(&mut wormhole_state, ctx(&mut test));
            init_and_share_state(
               bridge_state,
               my_emitter,
               12,
               x"1234567899012345678990123456789912",
               ctx(&mut test)
            );
            return_to_address<State>(admin, wormhole_state);
         };

        //test BridgeState setter and getter functions
        next_tx(&mut test, admin); {
            let state = take_shared<BridgeState>(&test);

            // test store consumed vaa
            bridge_state::store_consumed_vaa(&mut state, x"1234", ctx(&mut test));
            assert!(bridge_state::vaa_is_consumed(&state, x"1234") == true, 0);

            // TODO - test store coin store
            // TODO - test store treasury cap

            // test set governance chain id
            bridge_state::test_set_governance_chain_id(&mut state, u16::from_u64(5));
            assert!(bridge_state::get_governance_chain_id(&state) == u16::from_u64(5), 0);

            return_shared<BridgeState>(state);
        };
        test_scenario::end(test);
    }
}