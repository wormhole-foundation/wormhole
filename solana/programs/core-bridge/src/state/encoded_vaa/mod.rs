mod anchor;
pub use anchor::*;

mod zero_copy;
pub use zero_copy::*;

use anchor_lang::{prelude::*, Discriminator};

pub(crate) const VAA_BUF_START: usize = 8 // DISCRIMINATOR
    + crate::state::Header::INIT_SPACE
    + 4 // bytes.len()
    ;

pub(crate) const DISCRIMINATOR: [u8; 8] = anchor::EncodedVaa::DISCRIMINATOR;

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum ProcessingStatus {
    #[default]
    Unset,
    Writing,
    Verified,
}
