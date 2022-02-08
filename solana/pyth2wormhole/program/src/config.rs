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
    /// How many product/price pairs can be sent and attested at once
    pub max_batch_size: u16,
}

impl Owned for Pyth2WormholeConfig {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub type P2WConfigAccount<'b, const IsInitialized: AccountState> =
    Derive<Data<'b, Pyth2WormholeConfig, { IsInitialized }>, "pyth2wormhole-config">;
