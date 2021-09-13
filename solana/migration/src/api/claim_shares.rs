use crate::{
    accounts::{
        AuthoritySigner,
        CustodySigner,
        FromCustodyTokenAccountDerivationData,
        MigrationPool,
        ShareMint,
        ShareMintDerivationData,
        ToCustodyTokenAccount,
    },
    types::SplAccount,
    MigrationError::WrongMint,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

use crate::accounts::MigrationPoolDerivationData;
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    *,
};

#[derive(FromAccounts)]
pub struct ClaimShares<'b> {
    pub pool: Mut<MigrationPool<'b, { AccountState::Initialized }>>,
    pub from_token_custody: Mut<ToCustodyTokenAccount<'b, { AccountState::Initialized }>>,
    pub share_mint: Mut<ShareMint<'b, { AccountState::Initialized }>>,

    pub from_lp_acc: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub lp_share_acc: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub custody_signer: CustodySigner<'b>,
    pub authority_signer: AuthoritySigner<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct ClaimSharesData {
    pub amount: u64,
}

pub fn claim_shares(
    ctx: &ExecutionContext,
    accs: &mut ClaimShares,
    data: ClaimSharesData,
) -> Result<()> {
    if accs.lp_share_acc.mint != *accs.share_mint.info().key {
        return Err(WrongMint.into());
    }
    accs.from_token_custody.verify_derivation(
        ctx.program_id,
        &FromCustodyTokenAccountDerivationData {
            pool: *accs.pool.info().key,
        },
    )?;
    accs.share_mint.verify_derivation(
        ctx.program_id,
        &ShareMintDerivationData {
            pool: *accs.pool.info().key,
        },
    )?;
    accs.pool.verify_derivation(
        ctx.program_id,
        &MigrationPoolDerivationData {
            from: accs.pool.from,
            to: accs.pool.to,
        },
    )?;

    // Transfer claimed tokens to LP
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.from_token_custody.info().key,
        accs.from_lp_acc.info().key,
        accs.custody_signer.key,
        &[],
        data.amount,
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    // Burn LP shares
    let mint_ix = spl_token::instruction::burn(
        &spl_token::id(),
        accs.lp_share_acc.info().key,
        accs.share_mint.info().key,
        accs.authority_signer.key,
        &[],
        data.amount,
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.authority_signer, None)?;

    Ok(())
}
