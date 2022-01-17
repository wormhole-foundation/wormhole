use crate::{
    accounts::{
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::{
        PayloadTransfer,
        PayloadTransferWithPayload,
    },
    types::*,
    TokenBridgeError::*,
};
use bridge::{
    vaa::ClaimableVAA,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};
use spl_token::state::{
    Account,
    Mint,
};
use std::ops::{
    Deref,
    DerefMut,
};

#[derive(FromAccounts)]
pub struct CompleteNativeWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub vaa: ClaimableVAA<'b, PayloadTransferWithPayload>,
    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    /// Transfer with payload can only be redeemed by the recipient. The idea is
    /// to target contracts which can then decide how to process the payload.
    ///
    /// The actual recipient (the `to` field above) is an associated token
    /// account of the target contract and not the contract itself, so we also need
    /// to take the target contract's address directly. This will be the owner
    /// of the associated token account. This ownership check cannot be
    /// expressed in Solitaire, so we have to check it explicitly in
    /// [`complete_native_with_payload`]
    /// We require that the contract is a signer of this transaction.
    pub to_owner: MaybeMut<Signer<Info<'b>>>,
    pub to_fees: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub custody: Mut<CustodyAccount<'b, { AccountState::Initialized }>>,
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,

    pub custody_signer: CustodySigner<'b>,
}

impl<'a> From<&CompleteNativeWithPayload<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteNativeWithPayload<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteNativeWithPayload<'a>> for CustodyAccountDerivationData {
    fn from(accs: &CompleteNativeWithPayload<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

impl<'b> InstructionContext<'b> for CompleteNativeWithPayload<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteNativeWithPayloadData {}

pub fn complete_native_with_payload(
    ctx: &ExecutionContext,
    accs: &mut CompleteNativeWithPayload,
    data: CompleteNativeWithPayloadData,
) -> Result<()> {
    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify that the custody account is derived correctly
    let derivation_data: CustodyAccountDerivationData = (&*accs).into();
    accs.custody
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mints
    if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.to_fees.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.custody.mint {
        return Err(InvalidMint.into());
    }
    if *accs.custody_signer.key != accs.custody.owner {
        return Err(WrongAccountOwner.into());
    }

    // Verify VAA
    if accs.vaa.token_address != accs.mint.info().key.to_bytes() {
        return Err(InvalidMint.into());
    }
    if accs.vaa.token_chain != 1 {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to != accs.to_owner.info().key.to_bytes() {
        return Err(InvalidRecipient.into());
    }

    // VAA-specified recipient must be token account owner
    if *accs.to_owner.info().key != accs.to.owner {
        return Err(InvalidRecipient.into());
    }

    // Prevent vaa double signing
    accs.vaa.verify(ctx.program_id)?;
    accs.vaa.claim(ctx, accs.payer.key)?;

    let mut amount = accs.vaa.amount.as_u64();
    let mut fee = accs.vaa.fee.as_u64();

    // Wormhole always caps transfers at 8 decimals; un-truncate if the local token has more
    if accs.mint.decimals > 8 {
        amount *= 10u64.pow((accs.mint.decimals - 8) as u32);
        fee *= 10u64.pow((accs.mint.decimals - 8) as u32);
    }

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.custody.info().key,
        accs.to.info().key,
        accs.custody_signer.key,
        &[],
        amount.checked_sub(fee).unwrap(),
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    // Transfer fees
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.custody.info().key,
        accs.to_fees.info().key,
        accs.custody_signer.key,
        &[],
        fee,
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    Ok(())
}

#[derive(FromAccounts)]
pub struct CompleteWrappedWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    // Signed message for the transfer
    pub vaa: ClaimableVAA<'b, PayloadTransferWithPayload>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    /// Transfer with payload can only be redeemed by the recipient. The idea is
    /// to target contracts which can then decide how to process the payload.
    ///
    /// The actual recipient (the `to` field above) is an associated token
    /// account of the target contract and not the contract itself, so we also need
    /// to take the target contract's address directly. This will be the owner
    /// of the associated token account. This ownership check cannot be
    /// expressed in Solitaire, so we have to check it explicitly in
    /// [`complete_native_with_payload`]
    /// We require that the contract is a signer of this transaction.
    pub to_owner: MaybeMut<Signer<Info<'b>>>,
    pub to_fees: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::Initialized }>>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CompleteWrappedWithPayload<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteWrappedWithPayload<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteWrappedWithPayload<'a>> for WrappedDerivationData {
    fn from(accs: &CompleteWrappedWithPayload<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
        }
    }
}

impl<'b> InstructionContext<'b> for CompleteWrappedWithPayload<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteWrappedWithPayloadData {}

pub fn complete_wrapped_with_payload(
    ctx: &ExecutionContext,
    accs: &mut CompleteWrappedWithPayload,
    data: CompleteWrappedWithPayloadData,
) -> Result<()> {
    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mint
    accs.wrapped_meta.verify_derivation(
        ctx.program_id,
        &WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        },
    )?;
    if accs.wrapped_meta.token_address != accs.vaa.token_address
        || accs.wrapped_meta.chain != accs.vaa.token_chain
    {
        return Err(InvalidMint.into());
    }

    // Verify mints
    if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.to_fees.mint {
        return Err(InvalidMint.into());
    }

    // Verify VAA
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to != accs.to_owner.info().key.to_bytes() {
        return Err(InvalidRecipient.into());
    }

    // VAA-specified recipient must be token account owner
    if *accs.to_owner.info().key != accs.to.owner {
        return Err(InvalidRecipient.into());
    }

    accs.vaa.verify(ctx.program_id)?;
    accs.vaa.claim(ctx, accs.payer.key)?;

    // Mint tokens
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        accs.mint.info().key,
        accs.to.info().key,
        accs.mint_authority.key,
        &[],
        accs.vaa
            .amount
            .as_u64()
            .checked_sub(accs.vaa.fee.as_u64())
            .unwrap(),
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.mint_authority, None)?;

    // Mint fees
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        accs.mint.info().key,
        accs.to_fees.info().key,
        accs.mint_authority.key,
        &[],
        accs.vaa.fee.as_u64(),
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.mint_authority, None)?;

    Ok(())
}
