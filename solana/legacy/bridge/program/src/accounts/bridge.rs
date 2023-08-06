//! The Bridge account contains the main state for the wormhole bridge, as well as tracking
//! configuration options for how the bridge should behave.

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solitaire::{
    AccountOwner,
    AccountState,
    Data,
    Derive,
    Owned,
};

pub type Bridge<'a, const State: AccountState> = Derive<Data<'a, BridgeData, { State }>, "Bridge">;

#[derive(Clone, Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
pub struct BridgeData {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,

    /// Lamports in the collection account
    pub last_lamports: u64,

    /// Bridge configuration, which is set once upon initialization.
    pub config: BridgeConfig,
}

#[cfg(not(feature = "cpi"))]
impl Owned for BridgeData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for BridgeData {
    fn owner(&self) -> AccountOwner {
        use solana_program::pubkey::Pubkey;
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("BRIDGE_ADDRESS")).unwrap())
    }
}

#[derive(Clone, Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
pub struct BridgeConfig {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_expiration_time: u32,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee: u64,
}
