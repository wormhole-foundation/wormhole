//! Account schemas for the Core Bridge Program. This re-exports [mod@crate::legacy::state].

pub use crate::legacy::state::*;

mod encoded_vaa;
pub use encoded_vaa::*;
