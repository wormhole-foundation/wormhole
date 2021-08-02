use borsh::{BorshDeserialize, BorshSerialize};
use solana_program::pubkey::Pubkey;
use solitaire::{processors::seeded::AccountOwner, AccountState, Data, Derive, Owned};

#[derive(Default, BorshDeserialize, BorshSerialize)]
pub struct Pyth2WormholeConfig {
    ///  Authority owning this contract
    pub owner: Pubkey,
    /// Wormhole bridge program
    pub wh_prog: Pubkey,
    /// Authority owning Pyth price data
    pub pyth_owner: Pubkey,
}

impl Owned for Pyth2WormholeConfig {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub type P2WConfigAccount<'b, const IsInitialized: AccountState> =
    Derive<Data<'b, Pyth2WormholeConfig, { IsInitialized }>, "pyth2wormhole-config">;
