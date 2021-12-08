//! Exposes an API implementation depending on which feature flags have been toggled for the
//! library. Check submodules for chain runtime specific documentation.


#[cfg(feature = "solana")]
pub mod solana;
#[cfg(feature = "solana")]
pub use solana::*;


#[cfg(feature = "terra")]
pub mod terra;
#[cfg(feature = "terra")]
pub use terra::*;
