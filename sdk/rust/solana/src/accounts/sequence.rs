//! Account that tracks a forever-increasing nonce for a Wormhole contract.

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
pub struct Sequence(u64);

impl Account for Sequence {
    type Seeds = Pubkey;
    type Output = Pubkey;

    fn key(id: &Pubkey, emitter: Pubkey) -> Pubkey {
        Pubkey::find_program_address(&[b"Sequence", emitter.as_ref()], id).0
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        Ok(Sequence(account.lamports()))
    }
}
