use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    pack_type,
    processors::seeded::{
        AccountOwner,
        Owned,
        SingleOwned,
    },
};
use spl_token::state::{
    Account,
    Mint,
};

pub type Address = [u8; 32];
pub type ChainID = u16;

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct Config {
    pub wormhole_bridge: Pubkey,
}

#[cfg(not(feature = "cpi"))]
impl Owned for Config {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for Config {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct EndpointRegistration {
    pub chain: ChainID,
    pub contract: Address,
}

#[cfg(not(feature = "cpi"))]
impl Owned for EndpointRegistration {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

impl SingleOwned for EndpointRegistration {
}

#[cfg(feature = "cpi")]
impl Owned for EndpointRegistration {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct WrappedMeta {
    pub chain: ChainID,
    pub token_address: Address,
    pub original_decimals: u8,
}

impl SingleOwned for WrappedMeta {
}

#[cfg(not(feature = "cpi"))]
impl Owned for WrappedMeta {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for WrappedMeta {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

pub mod spl_token_2022 {
    use solana_program::pubkey::Pubkey;
    use std::str::FromStr;

    pub fn id() -> Pubkey {
        Pubkey::from_str("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb").unwrap()
    }
}

pack_type!(
    SplMint,
    Mint,
    AccountOwner::OneOf(vec![spl_token::id(), spl_token_2022::id()])
);
pack_type!(
    SplAccount,
    Account,
    AccountOwner::OneOf(vec![spl_token::id(), spl_token_2022::id()])
);
