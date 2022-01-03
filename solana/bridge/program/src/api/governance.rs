use solitaire::*;

use solana_program::{
    log::sol_log,
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
    sysvar::{
        clock::Clock,
        rent::Rent,
    },
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};

use crate::{
    accounts::{
        Bridge,
        GuardianSet,
        GuardianSetDerivationData,
    },
    error::Error::{
        InvalidFeeRecipient,
        InvalidGovernanceKey,
        InvalidGovernanceWithdrawal,
        InvalidGuardianSetUpgrade,
    },
    types::{
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        GovernancePayloadTransferFees,
        GovernancePayloadUpgrade,
    },
    vaa::ClaimableVAA,
    DeserializePayload,
    CHAIN_ID_SOLANA,
};

fn verify_governance<'a, T>(vaa: &ClaimableVAA<'a, T>) -> Result<()>
where
    T: DeserializePayload,
{
    let expected_emitter = std::option_env!("EMITTER_ADDRESS").ok_or_else(|| {
        sol_log("EMITTER_ADDRESS not set at compile-time");
        ProgramError::UninitializedAccount
    })?;
    let current_emitter = format!(
        "{}",
        Pubkey::new_from_array(vaa.message.meta().emitter_address)
    );
    // Fail if the emitter is not the known governance key, or the emitting chain is not Solana.
    if expected_emitter != current_emitter || vaa.message.meta().emitter_chain != CHAIN_ID_SOLANA {
        Err(InvalidGovernanceKey.into())
    } else {
        Ok(())
    }
}

#[derive(FromAccounts)]
pub struct UpgradeContract<'b> {
    /// Payer for account creation (vaa-claim)
    pub payer: Mut<Signer<Info<'b>>>,

    /// Bridge config
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// GuardianSet change VAA
    pub vaa: ClaimableVAA<'b, GovernancePayloadUpgrade>,

    /// PDA authority for the loader
    pub upgrade_authority: Derive<Info<'b>, "upgrade">,

    /// Spill address for the upgrade excess lamports
    pub spill: Mut<Info<'b>>,

    /// New contract address.
    pub buffer: Mut<Info<'b>>,

    /// Required by the upgradeable uploader.
    pub program_data: Mut<Info<'b>>,

    /// Our own address, required by the upgradeable loader.
    pub own_address: Mut<Info<'b>>,

    // Various sysvar/program accounts needed for the upgradeable loader.
    pub rent: Sysvar<'b, Rent>,
    pub clock: Sysvar<'b, Clock>,
    pub bpf_loader: Info<'b>,
    pub system: Info<'b>,
}

impl<'b> InstructionContext<'b> for UpgradeContract<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UpgradeContractData {}

pub fn upgrade_contract(
    ctx: &ExecutionContext,
    accs: &mut UpgradeContract,
    _data: UpgradeContractData,
) -> Result<()> {
    verify_governance(&accs.vaa)?;
    accs.vaa.verify(ctx.program_id)?;
    accs.vaa.claim(ctx, accs.payer.key)?;

    let upgrade_ix = solana_program::bpf_loader_upgradeable::upgrade(
        ctx.program_id,
        &accs.vaa.message.new_contract,
        accs.upgrade_authority.key,
        accs.spill.key,
    );

    let seeds = accs
        .upgrade_authority
        .self_bumped_seeds(None, ctx.program_id);
    let seeds: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
    let seeds = seeds.as_slice();
    invoke_signed(&upgrade_ix, ctx.accounts, &[seeds])?;

    Ok(())
}

#[derive(FromAccounts)]
pub struct UpgradeGuardianSet<'b> {
    /// Payer for account creation (vaa-claim)
    pub payer: Mut<Signer<Info<'b>>>,

    /// Bridge config
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// GuardianSet change VAA
    pub vaa: ClaimableVAA<'b, GovernancePayloadGuardianSetChange>,

    /// Old guardian set
    pub guardian_set_old: Mut<GuardianSet<'b, { AccountState::Initialized }>>,

    /// New guardian set
    pub guardian_set_new: Mut<GuardianSet<'b, { AccountState::Uninitialized }>>,
}

impl<'b> InstructionContext<'b> for UpgradeGuardianSet<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UpgradeGuardianSetData {}

pub fn upgrade_guardian_set(
    ctx: &ExecutionContext,
    accs: &mut UpgradeGuardianSet,
    _data: UpgradeGuardianSetData,
) -> Result<()> {
    // Enforce single increments when upgrading.
    if accs.guardian_set_old.index != accs.vaa.new_guardian_set_index - 1 {
        return Err(InvalidGuardianSetUpgrade.into());
    }

    // Confirm that the version the bridge has active is the previous version.
    if accs.bridge.guardian_set_index != accs.vaa.new_guardian_set_index - 1 {
        return Err(InvalidGuardianSetUpgrade.into());
    }

    verify_governance(&accs.vaa)?;
    accs.vaa.verify(ctx.program_id)?;
    accs.guardian_set_old.verify_derivation(
        ctx.program_id,
        &GuardianSetDerivationData {
            index: accs.vaa.new_guardian_set_index - 1,
        },
    )?;
    accs.guardian_set_new.verify_derivation(
        ctx.program_id,
        &GuardianSetDerivationData {
            index: accs.vaa.new_guardian_set_index,
        },
    )?;

    accs.vaa.claim(ctx, accs.payer.key)?;

    // Set expiration time for the old set
    accs.guardian_set_old.expiration_time =
        accs.vaa.meta().vaa_time + accs.bridge.config.guardian_set_expiration_time;

    // Initialize new guardian Set
    accs.guardian_set_new.index = accs.vaa.new_guardian_set_index;
    accs.guardian_set_new.creation_time = accs.vaa.meta().vaa_time;
    accs.guardian_set_new.keys = accs.vaa.new_guardian_set.clone();

    // Create new guardian set
    // This is done after populating it to properly allocate space according to key vec length.
    accs.guardian_set_new.create(
        &GuardianSetDerivationData {
            index: accs.guardian_set_new.index,
        },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    // Set guardian set index
    accs.bridge.guardian_set_index = accs.vaa.new_guardian_set_index;

    Ok(())
}

#[derive(FromAccounts)]
pub struct SetFees<'b> {
    /// Payer for account creation (vaa-claim)
    pub payer: Mut<Signer<Info<'b>>>,

    /// Bridge config
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// Governance VAA
    pub vaa: ClaimableVAA<'b, GovernancePayloadSetMessageFee>,
}

impl<'b> InstructionContext<'b> for SetFees<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct SetFeesData {}

pub fn set_fees(ctx: &ExecutionContext, accs: &mut SetFees, _data: SetFeesData) -> Result<()> {
    verify_governance(&accs.vaa)?;
    accs.vaa.verify(ctx.program_id)?;
    accs.vaa.claim(ctx, accs.payer.key)?;
    accs.bridge.config.fee = accs.vaa.fee.as_u64();

    Ok(())
}

#[derive(FromAccounts)]
pub struct TransferFees<'b> {
    /// Payer for account creation (vaa-claim)
    pub payer: Mut<Signer<Info<'b>>>,

    /// Bridge config
    pub bridge: Bridge<'b, { AccountState::Initialized }>,

    /// Governance VAA
    pub vaa: ClaimableVAA<'b, GovernancePayloadTransferFees>,

    /// Account collecting tx fees
    pub fee_collector: Mut<Derive<Info<'b>, "fee_collector">>,

    /// Fee recipient
    pub recipient: Mut<Info<'b>>,

    /// Rent calculator to check transfer sizes.
    pub rent: Sysvar<'b, Rent>,
}

impl<'b> InstructionContext<'b> for TransferFees<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferFeesData {}

pub fn transfer_fees(
    ctx: &ExecutionContext,
    accs: &mut TransferFees,
    _data: TransferFeesData,
) -> Result<()> {
    // Make sure the account loaded to receive funds is equal to the one the VAA requested.
    if accs.vaa.to != accs.recipient.key.to_bytes() {
        return Err(InvalidFeeRecipient.into());
    }

    verify_governance(&accs.vaa)?;
    accs.vaa.verify(ctx.program_id)?;

    if accs
        .fee_collector
        .lamports()
        .saturating_sub(accs.vaa.amount.as_u64())
        < accs.rent.minimum_balance(accs.fee_collector.data_len())
    {
        return Err(InvalidGovernanceWithdrawal.into());
    }

    accs.vaa.claim(ctx, accs.payer.key)?;

    // Transfer fees
    let transfer_ix = solana_program::system_instruction::transfer(
        accs.fee_collector.key,
        accs.recipient.key,
        accs.vaa.amount.as_u64(),
    );

    let seeds = accs.fee_collector.self_bumped_seeds(None, ctx.program_id);
    let seeds: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
    let seeds = seeds.as_slice();
    invoke_signed(&transfer_ix, ctx.accounts, &[seeds])?;

    Ok(())
}
