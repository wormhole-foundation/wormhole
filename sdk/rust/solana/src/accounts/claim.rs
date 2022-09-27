//! Claim accounts are PDA's used to prevent replay attacks.

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
    wormhole::{
        Chain,
        WormholeError,
    },
};

#[derive(Debug, Eq, PartialEq, BorshSerialize, BorshDeserialize)]
pub struct Claim {
    pub claimed: bool,
}

pub struct ClaimSeeds {
    pub emitter:  Pubkey,
    pub chain:    Chain,
    pub sequence: u64,
}

impl Account for Claim {
    type Seeds = ClaimSeeds;
    type Output = Pubkey;

    fn key(id: &Pubkey, seeds: Self::Seeds) -> Pubkey {
        Pubkey::find_program_address(
            &[
                seeds.emitter.as_ref(),
                &u16::from(seeds.chain).to_be_bytes(),
                &seeds.sequence.to_be_bytes(),
            ],
            id,
        )
        .0
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        Claim::try_from_slice(&account.data.borrow()).map_err(|_| WormholeError::DeserializeFailed)
    }
}
