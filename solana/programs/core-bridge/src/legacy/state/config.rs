use crate::types::Duration;
use anchor_lang::prelude::*;

/// Account used to store the current configuration of the bridge, including tracking Wormhole fee
/// payments. For governance decrees, the guardian set index is used to determine whether a decree
/// was attested for using the latest guardian set.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Config {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,

    /// Gap. In the old implementation, this was an amount that kept track of message fees that
    /// were paid to the program's fee collector.
    pub _gap_0: [u8; 8],

    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_ttl: Duration,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee_lamports: u64,
}

impl crate::legacy::utils::LegacyAccount for Config {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl Config {
    pub const SEED_PREFIX: &'static [u8] = b"Bridge";
}
