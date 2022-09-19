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
pub struct FeeCollector(u64);

impl Account for FeeCollector {
    type Seeds = ();
    type Output = Pubkey;

    fn key(id: &Pubkey, _: ()) -> Pubkey {
        let (fee_collector, _) = Pubkey::find_program_address(&[b"fee_collector"], id);
        fee_collector
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        Ok(FeeCollector(account.lamports()))
    }
}
