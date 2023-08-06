use crate::{
    accounts::{
        CustodySigner,
        FromCustodyTokenAccount,
        FromCustodyTokenAccountDerivationData,
        MigrationPool,
        MigrationPoolDerivationData,
        ShareMint,
        ShareMintDerivationData,
        ToCustodyTokenAccount,
        ToCustodyTokenAccountDerivationData,
    },
    types::SplMint,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::program::invoke_signed;
use solitaire::{
    CreationLamports::Exempt,
    *,
};

#[derive(FromAccounts)]
pub struct CreatePool<'b> {
    pub payer: Mut<Signer<Info<'b>>>,

    pub pool: Mut<MigrationPool<'b, { AccountState::Uninitialized }>>,
    pub from_mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub to_mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub from_token_custody: Mut<FromCustodyTokenAccount<'b, { AccountState::Uninitialized }>>,
    pub to_token_custody: Mut<ToCustodyTokenAccount<'b, { AccountState::Uninitialized }>>,
    pub pool_mint: Mut<ShareMint<'b, { AccountState::Uninitialized }>>,

    pub custody_signer: CustodySigner<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CreatePoolData {}

pub fn create_pool(
    ctx: &ExecutionContext,
    accs: &mut CreatePool,
    _data: CreatePoolData,
) -> Result<()> {
    // Create from custody account
    accs.from_token_custody.create(
        &FromCustodyTokenAccountDerivationData {
            pool: *accs.pool.info().key,
        },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    let init_ix = spl_token::instruction::initialize_account(
        &spl_token::id(),
        accs.from_token_custody.info().key,
        accs.from_mint.info().key,
        accs.custody_signer.info().key,
    )?;
    invoke_signed(&init_ix, ctx.accounts, &[])?;

    // Create to custody account
    accs.to_token_custody.create(
        &ToCustodyTokenAccountDerivationData {
            pool: *accs.pool.info().key,
        },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    let init_ix = spl_token::instruction::initialize_account(
        &spl_token::id(),
        accs.to_token_custody.info().key,
        accs.to_mint.info().key,
        accs.custody_signer.info().key,
    )?;
    invoke_signed(&init_ix, ctx.accounts, &[])?;

    // Create to pool mint
    accs.pool_mint.create(
        &ShareMintDerivationData {
            pool: *accs.pool.info().key,
        },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    let init_ix = spl_token::instruction::initialize_mint(
        &spl_token::id(),
        accs.pool_mint.info().key,
        accs.custody_signer.info().key,
        None,
        accs.from_mint.decimals,
    )?;
    invoke_signed(&init_ix, ctx.accounts, &[])?;

    // Set fields on pool
    accs.pool.from = *accs.from_mint.info().key;
    accs.pool.to = *accs.to_mint.info().key;

    // Create pool
    accs.pool.create(
        &MigrationPoolDerivationData {
            from: *accs.from_mint.info().key,
            to: *accs.to_mint.info().key,
        },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    Ok(())
}
