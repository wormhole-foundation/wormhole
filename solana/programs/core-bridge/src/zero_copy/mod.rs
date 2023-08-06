//! Zero-copy account structs for the Core Bridge Program. Wherever possible, it is recommended to
//! use these structs instead of using Anchor's [anchor_lang::prelude::Account] to deserialize these
//! accounts.

mod config;
pub use config::*;

mod encoded_vaa;
pub use encoded_vaa::*;

mod posted_message_v1;
pub use posted_message_v1::*;

mod posted_message_v1_unreliable;
pub use posted_message_v1_unreliable::*;

mod posted_vaa_v1;
pub use posted_vaa_v1::*;
