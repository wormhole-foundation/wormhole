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
        write_pauser_addresses,
        CONFIG_FULL_LEN,
    },
    TokenBridgeError::{
        InvalidGovernanceKey,
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

#[derive(FromAccounts)]
pub struct SetPauserAddresses<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    /// Existing token bridge `Config` PDA. Realloc'd from 32 → `CONFIG_FULL_LEN` bytes the first
    /// time this governance VAA is processed; subsequent VAAs only update the tail.
    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Governance VAA carrying a `PayloadSetPauserAddresses` (action 5).
    pub vaa: PayloadMessage<'b, PayloadSetPauserAddresses>,

    /// VAA replay-protection claim, consistent with the rest of the token bridge governance.
    pub claim: Mut<Claim<'b>>,
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

    // One-time migration: legacy 32-byte Config accounts grow to 97 bytes the first time this
    // VAA is processed. Subsequent VAAs reuse the already-extended layout. The `paused` flag
    // is preserved across updates — rotating the pauser/unpauser keys does not implicitly
    // unpause the bridge.
    let config_info = accs.config.info();
    if config_info.data_len() < CONFIG_FULL_LEN {
        // Top up the lamport balance so the account remains rent-exempt at the new size, then
        // realloc. `zero_init = true` zeroes the new tail bytes — that's the desired
        // initial state for `paused = false`.
        use solana_program::sysvar::Sysvar;
        let rent = solana_program::sysvar::rent::Rent::get()?;
        let required = rent.minimum_balance(CONFIG_FULL_LEN);
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
        config_info.realloc(CONFIG_FULL_LEN, true)?;
    }

    let mut data = config_info.data.borrow_mut();
    write_pauser_addresses(&mut data, &accs.vaa.pauser, &accs.vaa.unpauser);

    Ok(())
}
