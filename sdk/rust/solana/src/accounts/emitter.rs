//! Account that collects fees for the Wormhole contract.

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
pub struct Emitter(u64);

impl Account for Emitter {
    type Seeds = ();
    type Output = (Pubkey, Vec<&'static [u8]>, u8);

    fn key(id: &Pubkey, _: ()) -> (Pubkey, Vec<&'static [u8]>, u8) {
        let seeds: &[&[u8]] = &[b"emitter"];
        let (emitter, bump) = Pubkey::find_program_address(seeds, id);
        (emitter, seeds.to_vec(), bump)
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        Ok(Emitter(account.lamports()))
    }
}
