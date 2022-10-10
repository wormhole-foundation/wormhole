module token_bridge::bridge_state {
   //use std::vector::{Self};
   use std::option::{Self, Option};

   use sui::object::{UID};
   //use sui::coin::TreasuryCap;
   use sui::vec_map::{Self, VecMap};
   use sui::vec_set::{Self, VecSet};

   use wormhole::external_address::ExternalAddress;
   use wormhole::myu16::{U16};

   const E_ORIGIN_CHAIN_MISMATCH: u64 = 0;
   const E_ORIGIN_ADDRESS_MISMATCH: u64 = 1;
   const E_WRAPPING_NATIVE_COIN: u64 = 2;
   const E_WRAPPED_ASSET_NOT_INITIALIZED: u64 = 3;

   friend token_bridge::vaa;
   friend token_bridge::register_chain;

   /// The origin chain and address of a token.  In case of native tokens
   /// (where the chain is aptos), the token_address is the hash of the token
   /// info (see token_hash.move for more details)
   struct OriginInfo has store, copy, drop {
      token_chain: U16,
      token_address: ExternalAddress,
   }

   struct BridgeState has key {
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

   public (friend) fun set_registered_emitter(state: &mut BridgeState, chain_id: U16, emitter: ExternalAddress) {
      vec_map::insert<U16, ExternalAddress>(&mut state.registered_emitters, chain_id, emitter);
   }
}
