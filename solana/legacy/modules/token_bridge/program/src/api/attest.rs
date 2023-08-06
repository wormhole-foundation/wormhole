use crate::{
    accounts::{
        deserialize_and_verify_metadata,
        ConfigAccount,
        CoreBridge,
        EmitterAccount,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedMetaDerivationData,
        WrappedTokenMeta,
    },
    messages::PayloadAssetMeta,
    types::*,
};
use bridge::{
    api::PostMessageData,
    types::ConsistencyLevel,
    vaa::SerializePayload,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    account_info::AccountInfo,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::invoke,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    *,
};

#[derive(FromAccounts)]
pub struct AttestToken<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Mint to attest
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Uninitialized }>,

    /// SPL Metadata for the associated Mint
    pub spl_metadata: SplTokenMeta<'b>,

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

impl<'a> From<&AttestToken<'a>> for WrappedMetaDerivationData {
    fn from(accs: &AttestToken<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

impl<'a> From<&AttestToken<'a>> for SplTokenMetaDerivationData {
    fn from(accs: &AttestToken<'a>) -> Self {
        SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct AttestTokenData {
    pub nonce: u32,
}

pub fn attest_token(
    ctx: &ExecutionContext,
    accs: &mut AttestToken,
    data: AttestTokenData,
) -> Result<()> {
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

    // Create Asset Metadata
    let mut payload = PayloadAssetMeta {
        token_address: accs.mint.info().key.to_bytes(),
        token_chain: CHAIN_ID_SOLANA,
        decimals: accs.mint.decimals,
        symbol: "".to_string(),
        name: "".to_string(),
    };

    // Assign metadata if an SPL Metadata account exists for the SPL token in question.
    if !accs.spl_metadata.data_is_empty() {
        let metadata = deserialize_and_verify_metadata(&accs.spl_metadata, (&*accs).into())?;
        payload.name = metadata.data.name.clone();
        payload.symbol = metadata.data.symbol;
    }

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
