use crate::{
    accounts::{ConfigAccount, EmitterAccount},
    messages::{PayloadAssetMeta, PayloadTransfer},
    types::*,
};
use bridge::{
    api::{PostMessage, PostMessageData},
    vaa::SerializePayload,
    types::ConsistencyLevel,
};
use primitive_types::U256;
use solana_program::{
    account_info::AccountInfo,
    instruction::{AccountMeta, Instruction},
    program::{invoke, invoke_signed},
    program_error::ProgramError,
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::processors::seeded::invoke_seeded;
use solitaire::{CreationLamports::Exempt, *};
use spl_token::{
    error::TokenError::OwnerMismatch,
    state::{Account, Mint},
};
use std::ops::{Deref, DerefMut};

#[derive(FromAccounts)]
pub struct AttestToken<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Mint to attest
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,
    pub mint_meta: Data<'b, SplMint, { AccountState::MaybeInitialized }>,

    /// CPI Context
    pub bridge: Mut<Info<'b>>,

    /// Account to store the posted message
    pub message: Mut<Info<'b>>,

    /// Emitter of the VAA
    pub emitter: EmitterAccount<'b>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,
}

impl<'b> InstructionContext<'b> for AttestToken<'b> {
    fn verify(&self, _: &Pubkey) -> Result<()> {
        Ok(())
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
    let transfer_ix =
        solana_program::system_instruction::transfer(accs.payer.key, accs.fee_collector.key, 1000);
    invoke(&transfer_ix, ctx.accounts)?;

    let payload = PayloadAssetMeta {
        token_address: accs.mint.info().key.to_bytes(),
        token_chain: 1,
        decimals: accs.mint.decimals,
        symbol: "".to_string(), // TODO metadata
        name: "".to_string(),
    };

    if accs.mint_meta.is_initialized() {
        // Populate fields
    }

    let params = (bridge::instruction::Instruction::PostMessage, PostMessageData {
        nonce: data.nonce,
        payload: payload.try_to_vec()?,
        consistency_level: ConsistencyLevel::Confirmed,
    });

    let ix = Instruction::new_with_bytes(
        accs.config.wormhole_bridge,
        params.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.bridge.key, false),
            AccountMeta::new(*accs.message.key, false),
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
