#[cfg(feature = "fake")]
pub mod fake;
mod query;

pub use query::*;

#[no_mangle]
extern "C" fn requires_wormhole() {}
