#[cfg(test)]
#[macro_use]
extern crate lazy_static;

pub mod contract;
pub mod msg;
pub mod state;

#[cfg(all(target_arch = "wasm32", not(feature = "library")))]
cosmwasm_std::create_entry_points!(contract);
