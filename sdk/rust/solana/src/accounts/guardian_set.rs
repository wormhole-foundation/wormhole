//! Account containing a valid Wormhole guardian set.

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
pub struct GuardianSet {
    /// Index representing an incrementing version number for this guardian set.
    pub index:           u32,
    /// ETH style public keys
    pub keys:            Vec<[u8; 20]>,
    /// Timestamp representing the time this guardian became active.
    pub creation_time:   u32,
    /// Expiration time when VAAs issued by this set are no longer valid.
    pub expiration_time: u32,
}

impl Account for GuardianSet {
    type Seeds = u32;
    type Output = Pubkey;

    fn key(id: &Pubkey, index: u32) -> Pubkey {
        Pubkey::find_program_address(&[b"GuardianSet", &index.to_be_bytes()], id).0
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        GuardianSet::try_from_slice(&account.data.borrow())
            .map_err(|_| WormholeError::DeserializeFailed)
    }
}
