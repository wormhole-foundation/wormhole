#[cfg(test)]
extern crate lazy_static;

pub mod contract;
pub mod msg;
pub mod state;
pub mod token_address;

#[cfg(test)]
mod testing;

// Chain ID of Terra 2.0
pub const CHAIN_ID: u16 = 18;
