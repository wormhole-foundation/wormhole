use crate::{
    accounts::{
        deserialize_and_verify_metadata,
        AuthoritySigner,
        ConfigAccount,
        CoreBridge,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        EmitterAccount,
        MintSigner,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadTransfer,
    types::*,
    TokenBridgeError,
    TokenBridgeError::WrongAccountOwner,
};
use bridge::{
    api::PostMessageData,
    types::ConsistencyLevel,
    vaa::SerializePayload,
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
    /// SPL Metadata for the associated Mint
    pub spl_metadata: SplTokenMeta<'b>,

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

impl<'a> From<&TransferNative<'a>> for SplTokenMetaDerivationData {
    fn from(accs: &TransferNative<'a>) -> Self {
        SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferNativeData {
    pub nonce: u32,
    pub target_address: Address,
    pub target_chain: ChainID,
}

pub fn transfer_native(
    ctx: &ExecutionContext,
    accs: &mut TransferNative,
    data: TransferNativeData,
) -> Result<()> {
    // Verify that the custody account is derived correctly
    let derivation_data: CustodyAccountDerivationData = (&*accs).into();
    accs.custody
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mints
    if accs.from.mint != *accs.mint.info().key {
        return Err(TokenBridgeError::InvalidMint.into());
    }

    // Verify that the token is not a wrapped token
    if let COption::Some(mint_authority) = accs.mint.mint_authority {
        if mint_authority == MintSigner::key(None, ctx.program_id) {
            return Err(TokenBridgeError::TokenNotNative.into());
        }
    }

    if !accs.custody.is_initialized() {
        accs.custody
            .create(&(&*accs).into(), ctx, accs.payer.key, Exempt)?;

        let init_ix = spl_token::instruction::initialize_account(
            &spl_token::id(),
            accs.custody.info().key,
            accs.mint.info().key,
            accs.custody_signer.key,
        )?;
        invoke_signed(&init_ix, ctx.accounts, &[])?;
    }

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.from.info().key,
        accs.custody.info().key,
        accs.authority_signer.key,
        &[],
        1,
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.authority_signer, None)?;

    // Pay fee
    let transfer_ix = solana_program::system_instruction::transfer(
        accs.payer.key,
        accs.fee_collector.key,
        accs.bridge.config.fee,
    );
    invoke(&transfer_ix, ctx.accounts)?;

    let metadata = deserialize_and_verify_metadata(&accs.spl_metadata, (&*accs).into())?;

    // Post message
    // Given there is no tokenID equivalent on Solana and each distinct token address is translated
    // into a new contract on EVM based chains (which is costly), we use a static token_address
    // and encode the mint in the token_id.
    let payload = PayloadTransfer {
        token_address: [1u8; 32],
        token_chain: 1,
        to: data.target_address,
        to_chain: data.target_chain,
        symbol: metadata.data.symbol,
        name: metadata.data.name,
        uri: metadata.data.uri,
        token_id: U256::from_big_endian(&accs.mint.info().key.to_bytes()),
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

#[derive(FromAccounts)]
pub struct TransferWrapped<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub from: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub from_owner: MaybeMut<Signer<Info<'b>>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::Initialized }>>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,
    /// SPL Metadata for the associated Mint
    pub spl_metadata: SplTokenMeta<'b>,

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

impl<'a> From<&TransferWrapped<'a>> for WrappedMetaDerivationData {
    fn from(accs: &TransferWrapped<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

impl<'a> From<&TransferWrapped<'a>> for SplTokenMetaDerivationData {
    fn from(accs: &TransferWrapped<'a>) -> Self {
        SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferWrappedData {
    pub nonce: u32,
    pub target_address: Address,
    pub target_chain: ChainID,
}

pub fn transfer_wrapped(
    ctx: &ExecutionContext,
    accs: &mut TransferWrapped,
    data: TransferWrappedData,
) -> Result<()> {
    // Verify that the from account is owned by the from_owner
    if &accs.from.owner != accs.from_owner.key {
        return Err(WrongAccountOwner.into());
    }

    // Verify mints
    if accs.mint.info().key != &accs.from.mint {
        return Err(TokenBridgeError::InvalidMint.into());
    }

    // Verify that meta is correct
    let derivation_data: WrappedMetaDerivationData = (&*accs).into();
    accs.wrapped_meta
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Burn tokens
    let burn_ix = spl_token::instruction::burn(
        &spl_token::id(),
        accs.from.info().key,
        accs.mint.info().key,
        accs.authority_signer.key,
        &[],
        1,
    )?;
    invoke_seeded(&burn_ix, ctx, &accs.authority_signer, None)?;

    // Pay fee
    let transfer_ix = solana_program::system_instruction::transfer(
        accs.payer.key,
        accs.fee_collector.key,
        accs.bridge.config.fee,
    );

    invoke(&transfer_ix, ctx.accounts)?;

    // Enfoce wrapped meta to be uninitialized.
    let derivation_data: WrappedMetaDerivationData = (&*accs).into();
    accs.wrapped_meta
        .verify_derivation(ctx.program_id, &derivation_data)?;

    let metadata = deserialize_and_verify_metadata(&accs.spl_metadata, (&*accs).into())?;

    // Post message
    let payload = PayloadTransfer {
        token_address: accs.wrapped_meta.token_address,
        token_chain: accs.wrapped_meta.chain,
        token_id: U256(accs.wrapped_meta.token_id),
        to: data.target_address,
        to_chain: data.target_chain,
        symbol: metadata.data.symbol,
        name: metadata.data.name,
        uri: metadata.data.uri,
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
