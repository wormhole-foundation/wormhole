use crate::{
    accounts::{
        AuthoritySigner,
        CustodySigner,
        MigrationPool,
        MigrationPoolDerivationData,
        ShareMint,
        ShareMintDerivationData,
        ToCustodyTokenAccount,
        ToCustodyTokenAccountDerivationData,
    },
    types::{
        SplAccount,
        SplMint,
    },
    MigrationError::WrongMint,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    *,
};

#[derive(FromAccounts)]
pub struct RemoveLiquidity<'b> {
    pub pool: Mut<MigrationPool<'b, { AccountState::Initialized }>>,
    pub from_mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub to_mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub to_token_custody: Mut<ToCustodyTokenAccount<'b, { AccountState::Initialized }>>,
    pub share_mint: Mut<ShareMint<'b, { AccountState::Initialized }>>,

    pub to_lp_acc: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub lp_share_acc: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub custody_signer: CustodySigner<'b>,
    pub authority_signer: AuthoritySigner<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct RemoveLiquidityData {
    pub amount: u64,
}

pub fn remove_liquidity(
    ctx: &ExecutionContext,
    accs: &mut RemoveLiquidity,
    data: RemoveLiquidityData,
) -> Result<()> {
    if *accs.from_mint.info().key != accs.pool.from {
        return Err(WrongMint.into());
    }
    if *accs.to_mint.info().key != accs.pool.to {
        return Err(WrongMint.into());
    }
    if accs.lp_share_acc.mint != *accs.share_mint.info().key {
        return Err(WrongMint.into());
    }
    accs.to_token_custody.verify_derivation(
        ctx.program_id,
        &ToCustodyTokenAccountDerivationData {
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

    // The out amount needs to be decimal adjusted
    let out_amount = if accs.from_mint.decimals > accs.to_mint.decimals {
        data.amount
            .checked_div(10u64.pow((accs.from_mint.decimals - accs.to_mint.decimals) as u32))
            .ok_or(SolitaireError::InsufficientFunds)?
    } else {
        data.amount
            .checked_mul(10u64.pow((accs.to_mint.decimals - accs.from_mint.decimals) as u32))
            .ok_or(SolitaireError::InsufficientFunds)?
    };

    // Transfer removed liquidity to LP
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.to_token_custody.info().key,
        accs.to_lp_acc.info().key,
        accs.custody_signer.key,
        &[],
        out_amount,
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
