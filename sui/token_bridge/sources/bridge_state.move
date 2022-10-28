module token_bridge::bridge_state {
   //use std::vector::{Self};
   use std::option::{Self, Option};

   use sui::object::{UID};
   use sui::vec_map::{Self, VecMap};
   use sui::vec_set::{Self, VecSet};
   use sui::dynamic_object_field::{Self};
   use sui::tx_context::{TxContext};

   use token_bridge::treasury::{Self, CoinStore};

   use wormhole::external_address::ExternalAddress;
   use wormhole::myu16::{U16};

   const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
   const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
   const E_WRAPPING_NATIVE_COIN: u64 = 2;
   const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

   friend token_bridge::vaa;
   friend token_bridge::register_chain;
   friend token_bridge::wrapped;
   friend token_bridge::complete_transfer;

   /// The origin chain and address of a token.  In case of native tokens
   /// (where the chain is aptos), the token_address is the hash of the token
   /// info (see token_hash.move for more details)
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

   struct BridgeState has key, store {
      id: UID,
      governance_chain_id: U16,
      governance_contract: ExternalAddress,

      /// Set of consumed VAA hashes
      consumed_vaas: VecSet<vector<u8>>,

      /// Track treasury caps IDs, which are mutably shared
      // treasury_cap_stores: VecMap<OriginInfo, &UID>,

      // Mapping of bridge contracts on other chains
      registered_emitters: VecMap<U16, ExternalAddress>,
   }

   // getters

   public fun vaa_is_consumed(state: &BridgeState, hash: vector<u8>): bool {
      vec_set::contains(&state.consumed_vaas, &hash)
   }

   public fun governance_chain_id(state: &BridgeState): U16 {
      state.governance_chain_id
   }

   public fun governance_contract(state: &BridgeState): ExternalAddress {
      state.governance_contract
   }

   // public fun wrapped_asset_info(native_info: OriginInfo): TypeInfo acquires State {
   //    let wrapped_infos = &borrow_global<State>(@token_bridge).wrapped_infos;
   //    let type_info = table::borrow(wrapped_infos, native_info).type_info;
   //    assert!(option::is_some(&type_info), E_WRAPPED_ASSET_NOT_INITIALIZED);
   //    option::extract(&mut type_info)
   // }

   // public fun native_asset_info(token_address: TokenHash): TypeInfo acquires State {
   //    let native_infos = &borrow_global<State>(@token_bridge).native_infos;
   //    *table::borrow(native_infos, token_address)
   // }

   /// Returns the origin information for a CoinType
   // public fun origin_info<CoinType>(): OriginInfo acquires OriginInfo {
   //    if (is_wrapped_asset<CoinType>()) {
   //       *borrow_global<OriginInfo>(type_info::account_address(&type_of<CoinType>()))
   //    } else {
   //       let token_chain = state::get_chain_id();
   //       let token_address = token_hash::get_external_address(&token_hash::derive<CoinType>());
   //       OriginInfo { token_chain, token_address }
   //    }
   // }


   public fun get_registered_emitter(state: &BridgeState, chain_id: &U16): Option<ExternalAddress> {
      if (vec_map::contains(&state.registered_emitters, chain_id)) {
         option::some(*vec_map::get(&state.registered_emitters, chain_id))
      } else {
         option::none()
      }
   }

   public(friend) fun deposit<CoinType>(
      bridge_state: &mut BridgeState,
      coin: Coin<CoinType>,
      origin_info: OriginInfo,
      ctx: &mut TxContext
   ) {

      // TODO: confirm that CoinStore<CoinType> exists as a child object of bridge_state
      //       if it is not a child object, initialize a CoinStore and transfer it to bridge
      //       if it is, obtain a reference to it

      let coin_store = dynamic_object_field::borrow_mut<OriginInfo, CoinStore>(state.id, origin_info);
      treasury::deposit<CoinType>(coin_store, coin, ctx);
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

      let coin_store = dynamic_object_field::borrow_mut<OriginInfo, CoinStore>(state.id, origin_info);
      let coins = treasury::withdraw<CoinType>(coin_store, value, ctx);
      return coins
      //let balance_mut = coins::balance_mut<CoinType>(&mut store.coins, ctx);
      //coin::take<CoinType>(balance_mut, value, ctx)
      //}
   }

   public(friend) fun mint<CoinType>(
      origin_info: OriginInfo,
      state: &mut BridgeState,
      recipient: address,
      value: u64,
      ctx: &mut TxContext,
   ): Coin<CoinType> {
      let treasury_cap_store = dynamic_object_field::borrow_mut<OriginInfo, TreasuryCapStore>(state.id, origin_info);
      let coins = treasury::mint<CoinType>(treasury_cap_store, value, ctx);
      return coins
   }

   // setters

   public(friend) fun set_vaa_consumed(state: &mut BridgeState, hash: vector<u8>) {
      vec_set::insert<vector<u8>>(&mut state.consumed_vaas, hash);
   }

   public(friend) fun set_governance_chain_id(state: &mut BridgeState, governance_chain_id: U16) {
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

}
