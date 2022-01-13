use crate::{
    config::P2WConfigAccount,
    types::PriceAttestation,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::{
    clock::Clock,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::{
        invoke,
        invoke_signed,
    },
    program_error::ProgramError,
    pubkey::Pubkey,
    rent::Rent,
};

use bridge::{
    accounts::BridgeData,
    types::ConsistencyLevel,
    PostMessageData,
};

use solitaire::{
    trace,
    AccountState,
    Derive,
    ExecutionContext,
    FromAccounts,
    Info,
    InstructionContext,
    Keyed,
    Mut,
    Peel,
    Result as SoliResult,
    Seeded,
    invoke_seeded,
    Signer,
    SolitaireError,
    Sysvar,
    ToInstruction,
};

pub type P2WEmitter<'b> = Derive<Info<'b>, "p2w-emitter">;

#[derive(FromAccounts, ToInstruction)]
pub struct Attest<'b> {
    // Payer also used for wormhole
    pub payer: Mut<Signer<Info<'b>>>,
    pub system_program: Info<'b>,
    pub config: P2WConfigAccount<'b, { AccountState::Initialized }>,
    pub pyth_product: Info<'b>,
    pub pyth_price: Info<'b>,
    pub clock: Sysvar<'b, Clock>,

    // post_message accounts
    /// Wormhole program address
    pub wh_prog: Info<'b>,

    /// Bridge config needed for fee calculation
    pub wh_bridge: Mut<Info<'b>>,

    /// Account to store the posted message
    pub wh_message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub wh_emitter: P2WEmitter<'b>,

    /// Tracker for the emitter sequence
    pub wh_sequence: Mut<Info<'b>>,

    // We reuse our payer
    // pub wh_payer: Mut<Signer<Info<'b>>>,
    /// Account to collect tx fee
    pub wh_fee_collector: Mut<Info<'b>>,

    pub wh_rent: Sysvar<'b, Rent>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct AttestData {
    pub nonce: u32,
    pub consistency_level: ConsistencyLevel,
}

impl<'b> InstructionContext<'b> for Attest<'b> {
    fn deps(&self) -> Vec<Pubkey> {
        vec![solana_program::system_program::id()]
    }
}

pub fn attest(ctx: &ExecutionContext, accs: &mut Attest, data: AttestData) -> SoliResult<()> {
    accs.config.verify_derivation(ctx.program_id, None)?;

    if accs.config.pyth_owner != *accs.pyth_price.owner
        || accs.config.pyth_owner != *accs.pyth_product.owner
    {
        trace!(&format!(
            "pyth_owner pubkey mismatch (expected {:?}, got price owner {:?} and product owner {:?}",
            accs.config.pyth_owner, accs.pyth_price.owner, accs.pyth_product.owner
        ));
        return Err(SolitaireError::InvalidOwner(accs.pyth_price.owner.clone()).into());
    }

    if accs.config.wh_prog != *accs.wh_prog.key {
        trace!(&format!(
            "Wormhole program account mismatch (expected {:?}, got {:?})",
            accs.config.wh_prog, accs.wh_prog.key
        ));
    }

    let price_attestation = PriceAttestation::from_pyth_price_bytes(
        accs.pyth_price.key.clone(),
        accs.clock.unix_timestamp,
        &*accs.pyth_price.try_borrow_data()?,
    )?;

    if &price_attestation.product_id != accs.pyth_product.key {
        trace!(&format!(
            "Price's product_id does not match the pased account (points at {:?} instead)",
            price_attestation.product_id
        ));
        return Err(ProgramError::InvalidAccountData.into());
    }

    let bridge_config = BridgeData::try_from_slice(&accs.wh_bridge.try_borrow_mut_data()?)?.config;

    // Pay wormhole fee
    let transfer_ix = solana_program::system_instruction::transfer(
        accs.payer.key,
        accs.wh_fee_collector.info().key,
        bridge_config.fee,
    );
    solana_program::program::invoke(&transfer_ix, ctx.accounts)?;

    // Send payload
    let post_message_data = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
            nonce: data.nonce,
            payload: price_attestation.serialize(),
            consistency_level: data.consistency_level,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wh_prog,
        post_message_data.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.wh_bridge.key, false),
            AccountMeta::new_readonly(*accs.wh_message.key, true),
            AccountMeta::new_readonly(*accs.wh_emitter.key, true),
            AccountMeta::new(*accs.wh_sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.wh_fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
    );

    trace!("Before cross-call");

    invoke_seeded(&ix, ctx, &accs.wh_emitter, None)?;

    Ok(())
}
