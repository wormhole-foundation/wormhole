#![allow(incomplete_features)]

#![feature(adt_const_params)]

use api::{
    add_liquidity::*,
    claim_shares::*,
    create_pool::*,
    migrate_tokens::*,
    remove_liquidity::*,
};
use solitaire::{
    solitaire,
    SolitaireError,
};

pub mod accounts;
pub mod api;
pub mod types;

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
extern crate wasm_bindgen;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;

pub enum MigrationError {
    WrongMint,
}

impl From<MigrationError> for SolitaireError {
    fn from(t: MigrationError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    AddLiquidity(AddLiquidityData) => add_liquidity,
    RemoveLiquidity(RemoveLiquidityData) => remove_liquidity,
    ClaimShares(ClaimSharesData) => claim_shares,
    CreatePool(CreatePoolData) => create_pool,
    MigrateTokens(MigrateTokensData) => migrate_tokens,
}
