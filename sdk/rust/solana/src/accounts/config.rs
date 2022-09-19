//! Solana Config account.

use {
    super::Account,
    borsh::{
        BorshDeserialize,
        BorshSerialize,
    },
    solana_program::{
        account_info::AccountInfo,
        pubkey::Pubkey,
    },
    wormhole::WormholeError,
};

#[derive(Debug, Eq, PartialEq, BorshSerialize, BorshDeserialize)]
pub struct Params {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.
    pub guardian_set_expiration_time: u32,
    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee:                          u64,
}

#[derive(Debug, Eq, PartialEq, BorshSerialize, BorshDeserialize)]
pub struct Config {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,
    /// Lamports in the collection account
    pub last_lamports:      u64,
    /// Bridge params, which are set once upon contract initialization.
    pub params:             Params,
}

impl Account for Config {
    type Seeds = ();
    type Output = Pubkey;

    fn key(id: &Pubkey, _: ()) -> Pubkey {
        let (config, _) = Pubkey::find_program_address(&[b"Bridge"], id);
        config
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        Config::try_from_slice(&account.data.borrow()).map_err(|_| WormholeError::DeserializeFailed)
    }
}
