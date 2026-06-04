//! Instruction handlers.
//!
//! Only `submit_observations` is implemented. The sibling instructions and the
//! support modules it calls into (`noreplay`, `open_digest`, `pda_init`,
//! `transfer`) are documented stubs returning `NotImplemented`; each lands
//! with its own commit.

pub mod close_digest;
pub mod close_pending;
pub mod modify_balance;
pub mod noreplay;
pub(crate) mod open_digest;
pub mod pda_init;
pub mod register_chain;
pub mod submit_observations;
pub mod submit_vaas;
pub mod transfer;
