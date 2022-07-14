use crate::{
    accounts::{
        AuthoritySigner,
        ConfigAccount,
        CoreBridge,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        EmitterAccount,
        MintSigner,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadTransfer,
    types::*,
    TokenBridgeError,
    TokenBridgeError::{
        InvalidChain,
        InvalidFee,
        WrongAccountOwner,
    },
};
use bridge::{
    api::PostMessageData,
    types::ConsistencyLevel,
    vaa::SerializePayload,
    CHAIN_ID_SOLANA,
};
use primitive_types::U256;
use solana_program::{
    account_info::AccountInfo,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::{
        invoke,
        invoke_signed,
    },
    program_option::COption,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};

#[derive(FromAccounts)]
pub struct TransferNative<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub from: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,

    pub mint: Mut<Data<'b, SplMint, { AccountState::Initialized }>>,

    pub custody: Mut<CustodyAccount<'b, { AccountState::MaybeInitialized }>>,

    // This could allow someone to race someone else's tx if they do the approval in a separate tx.
    // Therefore the approval must be set in the same tx.
    pub authority_signer: AuthoritySigner<'b>,

    pub custody_signer: CustodySigner<'b>,

    /// CPI Context
    pub bridge: Mut<CoreBridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub emitter: EmitterAccount<'b>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,
}

impl<'a> From<&TransferNative<'a>> for CustodyAccountDerivationData {
    fn from(accs: &TransferNative<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferNativeData {
    pub nonce: u32,
    pub amount: u64,
    pub fee: u64,
    pub target_address: Address,
    pub target_chain: ChainID,
}

pub fn transfer_native(
    ctx: &ExecutionContext,
    accs: &mut TransferNative,
    data: TransferNativeData,
) -> Result<()> {
    // Prevent transferring to the same chain.
    if data.target_chain == CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let derivation_data: CustodyAccountDerivationData = (&*accs).into();
    let (amount, fee) = verify_and_execute_native_transfers(
        ctx,
        &derivation_data,
        &accs.payer,
        &accs.from,
        &accs.mint,
        &accs.custody,
        &accs.authority_signer,
        &accs.custody_signer,
        &accs.bridge,
        &accs.fee_collector,
        data.amount,
        data.fee,
    )?;

    // Post message
    let payload = PayloadTransfer {
        amount: U256::from(amount),
        token_address: accs.mint.info().key.to_bytes(),
        token_chain: CHAIN_ID_SOLANA,
        to: data.target_address,
        to_chain: data.target_chain,
        fee: U256::from(fee),
    };
    let params = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
            nonce: data.nonce,
            payload: payload.try_to_vec()?,
            consistency_level: ConsistencyLevel::Finalized,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wormhole_bridge,
        params.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.bridge.info().key, false),
            AccountMeta::new(*accs.message.key, true),
            AccountMeta::new_readonly(*accs.emitter.key, true),
            AccountMeta::new(*accs.sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
        ],
    );
    invoke_seeded(&ix, ctx, &accs.emitter, None)?;

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn verify_and_execute_native_transfers(
    ctx: &ExecutionContext,
    derivation_data: &CustodyAccountDerivationData,
    payer: &Mut<Signer<AccountInfo>>,
    from: &Mut<Data<SplAccount, { AccountState::Initialized }>>,
    mint: &Mut<Data<SplMint, { AccountState::Initialized }>>,
    custody: &Mut<CustodyAccount<{ AccountState::MaybeInitialized }>>,
    authority_signer: &AuthoritySigner,
    custody_signer: &CustodySigner,
    bridge: &Mut<CoreBridge<{ AccountState::Initialized }>>,
    fee_collector: &Mut<Info>,
    raw_amount: u64,
    raw_fee: u64,
) -> Result<(u64, u64)> {
    // Verify that the custody account is derived correctly
    custody.verify_derivation(ctx.program_id, derivation_data)?;

    // Verify mints
    if from.mint != *mint.info().key {
        return Err(TokenBridgeError::InvalidMint.into());
    }

    // Fee must be less than amount
    if raw_fee > raw_amount {
        return Err(InvalidFee.into());
    }

    // Verify that the token is not a wrapped token
    if let COption::Some(mint_authority) = mint.mint_authority {
        if mint_authority == MintSigner::key(None, ctx.program_id) {
            return Err(TokenBridgeError::TokenNotNative.into());
        }
    }

    if !custody.is_initialized() {
        custody.create(derivation_data, ctx, payer.key, Exempt)?;

        let init_ix = spl_token::instruction::initialize_account(
            &spl_token::id(),
            custody.info().key,
            mint.info().key,
            custody_signer.key,
        )?;
        invoke_signed(&init_ix, ctx.accounts, &[])?;
    }

    let trunc_divisor = 10u64.pow(8.max(mint.decimals as u32) - 8);
    // Truncate to 8 decimals
    let amount: u64 = raw_amount / trunc_divisor;
    let fee: u64 = raw_fee / trunc_divisor;
    // Untruncate the amount to drop the remainder so we don't  "burn" user's funds.
    let amount_trunc: u64 = amount * trunc_divisor;

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        from.info().key,
        custody.info().key,
        authority_signer.key,
        &[],
        amount_trunc,
    )?;
    invoke_seeded(&transfer_ix, ctx, authority_signer, None)?;

    // Pay fee
    let transfer_ix = solana_program::system_instruction::transfer(
        payer.key,
        fee_collector.key,
        bridge.config.fee,
    );
    invoke(&transfer_ix, ctx.accounts)?;

    Ok((amount, fee))
}

#[derive(FromAccounts)]
pub struct TransferWrapped<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub from: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub from_owner: MaybeMut<Signer<Info<'b>>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::Initialized }>>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,

    pub authority_signer: AuthoritySigner<'b>,

    /// CPI Context
    pub bridge: Mut<CoreBridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub emitter: EmitterAccount<'b>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,
}

impl<'a> From<&TransferWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &TransferWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: 1,
            token_address: accs.mint.info().key.to_bytes(),
        }
    }
}

impl<'a> From<&TransferWrapped<'a>> for WrappedMetaDerivationData {
    fn from(accs: &TransferWrapped<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferWrappedData {
    pub nonce: u32,
    pub amount: u64,
    pub fee: u64,
    pub target_address: Address,
    pub target_chain: ChainID,
}

pub fn transfer_wrapped(
    ctx: &ExecutionContext,
    accs: &mut TransferWrapped,
    data: TransferWrappedData,
) -> Result<()> {
    // Prevent transferring to the same chain.
    if data.target_chain == CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let derivation_data: WrappedMetaDerivationData = (&*accs).into();
    verify_and_execute_wrapped_transfers(
        ctx,
        &derivation_data,
        &accs.payer,
        &accs.from,
        &accs.from_owner,
        &accs.mint,
        &accs.wrapped_meta,
        &accs.authority_signer,
        &accs.bridge,
        &accs.fee_collector,
        data.amount,
        data.fee,
    )?;

    // Post message
    let payload = PayloadTransfer {
        amount: U256::from(data.amount),
        token_address: accs.wrapped_meta.token_address,
        token_chain: accs.wrapped_meta.chain,
        to: data.target_address,
        to_chain: data.target_chain,
        fee: U256::from(data.fee),
    };
    let params = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
            nonce: data.nonce,
            payload: payload.try_to_vec()?,
            consistency_level: ConsistencyLevel::Finalized,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wormhole_bridge,
        params.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.bridge.info().key, false),
            AccountMeta::new(*accs.message.key, true),
            AccountMeta::new_readonly(*accs.emitter.key, true),
            AccountMeta::new(*accs.sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
        ],
    );
    invoke_seeded(&ix, ctx, &accs.emitter, None)?;

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn verify_and_execute_wrapped_transfers(
    ctx: &ExecutionContext,
    derivation_data: &WrappedMetaDerivationData,
    payer: &Mut<Signer<AccountInfo>>,
    from: &Mut<Data<SplAccount, { AccountState::Initialized }>>,
    from_owner: &MaybeMut<Signer<Info>>,
    mint: &Mut<WrappedMint<{ AccountState::Initialized }>>,
    wrapped_meta: &WrappedTokenMeta<{ AccountState::Initialized }>,
    authority_signer: &AuthoritySigner,
    bridge: &Mut<CoreBridge<{ AccountState::Initialized }>>,
    fee_collector: &Mut<Info>,
    amount: u64,
    fee: u64,
) -> Result<()> {
    // Verify that the from account is owned by the from_owner
    if &from.owner != from_owner.key {
        return Err(WrongAccountOwner.into());
    }

    // Verify mints
    if mint.info().key != &from.mint {
        return Err(TokenBridgeError::InvalidMint.into());
    }

    // Fee must be less than amount
    if fee > amount {
        return Err(InvalidFee.into());
    }

    // Verify that meta is correct
    wrapped_meta.verify_derivation(ctx.program_id, derivation_data)?;

    // Burn tokens
    let burn_ix = spl_token::instruction::burn(
        &spl_token::id(),
        from.info().key,
        mint.info().key,
        authority_signer.key,
        &[],
        amount,
    )?;
    invoke_seeded(&burn_ix, ctx, authority_signer, None)?;

    // Pay fee
    let transfer_ix = solana_program::system_instruction::transfer(
        payer.key,
        fee_collector.key,
        bridge.config.fee,
    );

    invoke(&transfer_ix, ctx.accounts)?;

    Ok(())
}
