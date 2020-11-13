pub mod contract;
mod error;
pub mod msg;
pub mod state;

pub use crate::error::ContractError;

#[cfg(all(target_arch = "wasm32", not(feature = "library")))]
cosmwasm_std::create_entry_points!(contract);
