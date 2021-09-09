use crate::types::{
    PoolData,
    SplAccount,
    SplMint,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
    Data,
    Derive,
    Info,
};

pub type ShareMint<'a, const STATE: AccountState> = Data<'a, SplMint, { STATE }>;

pub struct ShareMintDerivationData {
    pub pool: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&ShareMintDerivationData> for ShareMint<'b, { STATE }> {
    fn seeds(accs: &ShareMintDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("share_mint").as_bytes().to_vec(),
            accs.pool.to_bytes().to_vec(),
        ]
    }
}

pub type FromCustodyTokenAccount<'a, const STATE: AccountState> = Data<'a, SplAccount, { STATE }>;

pub struct FromCustodyTokenAccountDerivationData {
    pub pool: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&FromCustodyTokenAccountDerivationData>
    for FromCustodyTokenAccount<'b, { STATE }>
{
    fn seeds(accs: &FromCustodyTokenAccountDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("from_custody").as_bytes().to_vec(),
            accs.pool.to_bytes().to_vec(),
        ]
    }
}

pub type ToCustodyTokenAccount<'a, const STATE: AccountState> = Data<'a, SplAccount, { STATE }>;

pub struct ToCustodyTokenAccountDerivationData {
    pub pool: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&ToCustodyTokenAccountDerivationData>
    for ToCustodyTokenAccount<'b, { STATE }>
{
    fn seeds(accs: &ToCustodyTokenAccountDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("to_custody").as_bytes().to_vec(),
            accs.pool.to_bytes().to_vec(),
        ]
    }
}

pub type MigrationPool<'a, const STATE: AccountState> = Data<'a, PoolData, { STATE }>;

pub struct MigrationPoolDerivationData {
    pub from: Pubkey,
    pub to: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&MigrationPoolDerivationData>
    for MigrationPool<'b, { STATE }>
{
    fn seeds(accs: &MigrationPoolDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("pool").as_bytes().to_vec(),
            accs.from.to_bytes().to_vec(),
            accs.to.to_bytes().to_vec(),
        ]
    }
}

pub type CustodySigner<'a> = Derive<Info<'a>, "custody_signer">;
pub type AuthoritySigner<'a> = Derive<Info<'a>, "authority_signer">;
