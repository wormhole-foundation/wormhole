use crate::{
    accounts::{
        ConfigAccount,
        Endpoint,
        EndpointDerivationData,
    },
    messages::{
        GovernancePayloadUpgrade,
        PayloadGovernanceRegisterChain,
        PayloadSetPauserAddresses,
    },
    types::{
        write_pause_authorities,
        CONFIG_WITH_PAUSER_LEN,
    },
    TokenBridgeError::{
        InvalidGovernanceKey,
        InvalidSelfProgram,
        InvalidVAA,
    },
    INVALID_VAAS,
};
use bridge::{
    accounts::claim::{
        self,
        Claim,
    },
    DeserializePayload,
    PayloadMessage,
    CHAIN_ID_GOVERNANCE,
};
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
    pubkey::Pubkey,
    sysvar::{
        clock::Clock,
        rent::Rent,
    },
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
    *,
};

// Confirm that a ClaimableVAA came from the correct chain, signed by the right emitter.
fn verify_governance<T>(vaa: &PayloadMessage<T>) -> Result<()>
where
    T: DeserializePayload,
{
    let expected_emitter = std::env!("EMITTER_ADDRESS");
    let current_emitter = format!("{}", Pubkey::new_from_array(vaa.meta().emitter_address));
    // Fail if the emitter is not the known governance key, or the emitting chain is not the governance chain.
    if expected_emitter != current_emitter || vaa.meta().emitter_chain != CHAIN_ID_GOVERNANCE {
        Err(InvalidGovernanceKey.into())
    } else {
        Ok(())
    }
}

#[derive(FromAccounts)]
pub struct UpgradeContract<'b> {
    /// Payer for account creation (vaa-claim)
    pub payer: Mut<Signer<Info<'b>>>,

    /// GuardianSet change VAA
    pub vaa: PayloadMessage<'b, GovernancePayloadUpgrade>,
    pub claim: Mut<Claim<'b>>,

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

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UpgradeContractData {}

pub fn upgrade_contract(
    ctx: &ExecutionContext,
    accs: &mut UpgradeContract,
    _data: UpgradeContractData,
) -> Result<()> {
    if INVALID_VAAS.contains(&&*accs.vaa.info().key.to_string()) {
        return Err(InvalidVAA.into());
    }

    verify_governance(&accs.vaa)?;
    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    let upgrade_ix = solana_program::bpf_loader_upgradeable::upgrade(
        ctx.program_id,
        &accs.vaa.new_contract,
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
pub struct RegisterChain<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub endpoint: Mut<Endpoint<'b, { AccountState::Uninitialized }>>,

    pub vaa: PayloadMessage<'b, PayloadGovernanceRegisterChain>,
    pub claim: Mut<Claim<'b>>,
}

impl<'a> From<&RegisterChain<'a>> for EndpointDerivationData {
    fn from(accs: &RegisterChain<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.chain,
            emitter_address: accs.vaa.endpoint_address,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct RegisterChainData {}

pub fn register_chain(
    ctx: &ExecutionContext,
    accs: &mut RegisterChain,
    _data: RegisterChainData,
) -> Result<()> {
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.endpoint
        .verify_derivation(ctx.program_id, &derivation_data)?;

    if INVALID_VAAS.contains(&&*accs.vaa.info().key.to_string()) {
        return Err(InvalidVAA.into());
    }

    // Claim VAA
    verify_governance(&accs.vaa)?;
    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    // Create endpoint
    accs.endpoint
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

    accs.endpoint.chain = accs.vaa.chain;
    accs.endpoint.contract = accs.vaa.endpoint_address;

    Ok(())
}

/// Event discriminator: `SHA256("event:PauserAddressesSet")[0..8]`. Emitted via Anchor-style
/// self-CPI when `set_pauser_addresses` succeeds. Payload (96 bytes): 32-byte pauser pubkey, then
/// the 32-byte freezer pubkey, then the 32-byte unpauser pubkey, in that order.
pub const PAUSER_ADDRESSES_SET_EVENT_DISCRIMINATOR: [u8; 8] =
    [0xb9, 0xcf, 0x4f, 0x8f, 0x6c, 0x76, 0xdf, 0x6d];

#[derive(FromAccounts)]
pub struct SetPauserAddresses<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    /// Existing token bridge `Config` PDA. Realloc'd from 32 → `CONFIG_WITH_PAUSER_LEN` bytes the first
    /// time this governance VAA is processed; subsequent VAAs only update the tail.
    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Governance VAA carrying a `PayloadSetPauserAddresses` (action 4 per whitepaper 0003).
    pub vaa: PayloadMessage<'b, PayloadSetPauserAddresses>,

    /// VAA replay-protection claim, consistent with the rest of the token bridge governance.
    pub claim: Mut<Claim<'b>>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct SetPauserAddressesData {}

pub fn set_pauser_addresses(
    ctx: &ExecutionContext,
    accs: &mut SetPauserAddresses,
    _data: SetPauserAddressesData,
) -> Result<()> {
    if INVALID_VAAS.contains(&&*accs.vaa.info().key.to_string()) {
        return Err(InvalidVAA.into());
    }

    verify_governance(&accs.vaa)?;
    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    // One-time migration: the Config account is created by the bridge `initialize` at its
    // original 32-byte size; the first time this VAA is processed it grows straight to the full
    // CONFIG_WITH_PAUSER_LEN (137) bytes. Subsequent VAAs reuse the already-extended layout. The
    // `paused` flag and `pause_expiry` are preserved across updates — rotating the
    // pauser/freezer/unpauser keys does not implicitly unpause the bridge.
    //
    // Borsh back-compat: the first 32 bytes of the account (the original `Config` struct) are
    // never moved or rewritten, and the `Config::BorshDeserialize` impl in `types.rs` is
    // hand-rolled to tolerate the trailing tail (instead of the derived impl that would reject
    // any unread bytes). Off-chain clients that deserialize Config via Borsh against the raw
    // account data continue to round-trip both before and after this migration. SDKs that read
    // pauser state must use the `paused()` / `pauser()` / `unpauser()` helpers in `types.rs`,
    // or similar, which check the account length and treat un-migrated accounts as unassigned.
    let config_info = accs.config.info();
    if config_info.data_len() < CONFIG_WITH_PAUSER_LEN {
        // Top up the lamport balance so the account remains rent-exempt at the new size, then
        // realloc. `zero_init = true` zeroes the new tail bytes — that's the desired
        // initial state for `paused = false`.
        use solana_program::sysvar::Sysvar;
        let rent = solana_program::sysvar::rent::Rent::get()?;
        let required = rent.minimum_balance(CONFIG_WITH_PAUSER_LEN);
        let current = config_info.lamports();
        if current < required {
            let topup = required - current;
            let topup_ix = solana_program::system_instruction::transfer(
                accs.payer.key,
                config_info.key,
                topup,
            );
            invoke_signed(&topup_ix, ctx.accounts, &[])?;
        }
        config_info.realloc(CONFIG_WITH_PAUSER_LEN, true)?;
    }

    // Invariant the tail writers rely on: the account is now exactly the full pauser layout. The
    // only legitimate sizes reaching this point are the 32-byte legacy account (just realloc'd
    // above) and an already-migrated 137-byte account. Assert it explicitly so a corrupted /
    // unexpected intermediate size reverts cleanly here instead of panicking on an out-of-bounds
    // slice inside `write_pause_authorities`.
    if config_info.data_len() != CONFIG_WITH_PAUSER_LEN {
        return Err(InvalidVAA.into());
    }

    {
        let mut data = config_info.data.borrow_mut();
        write_pause_authorities(
            &mut data,
            &accs.vaa.pauser,
            &accs.vaa.freezer,
            &accs.vaa.unpauser,
        );
        // `data` borrow dropped here so the self-CPI below can re-borrow the account.
    }

    if accs.self_program.key != ctx.program_id {
        return Err(InvalidSelfProgram.into());
    }
    let mut payload = [0u8; 96];
    payload[..32].copy_from_slice(&accs.vaa.pauser.to_bytes());
    payload[32..64].copy_from_slice(&accs.vaa.freezer.to_bytes());
    payload[64..].copy_from_slice(&accs.vaa.unpauser.to_bytes());
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &PAUSER_ADDRESSES_SET_EVENT_DISCRIMINATOR,
        &payload,
    )
}
