use crate::{
    error::CoreBridgeError,
    legacy::{instruction::EmptyArgs, utils::LegacyAnchorized},
    state::{Config, GuardianSet},
    types::Timestamp,
    zero_copy::{LoadZeroCopy, VaaAccount},
};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;

#[derive(Accounts)]
pub struct GuardianSetUpdate<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// For governance VAAs, we need to make sure that the current guardian set was used to attest
    /// for this governance decree.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    vaa: AccountInfo<'info>,

    /// CHECK: Account representing that a VAA has been consumed. Seeds are checked when
    /// [claim_vaa](crate::utils::vaa::claim_vaa) is called.
    #[account(mut)]
    claim: AccountInfo<'info>,

    /// Existing guardian set, whose guardian set index is the same one found in the [Config].
    #[account(
        mut,
        seeds = [
            GuardianSet::SEED_PREFIX,
            &config.guardian_set_index.to_be_bytes()
        ],
        bump,
    )]
    curr_guardian_set: Account<'info, LegacyAnchorized<0, GuardianSet>>,

    /// New guardian set created from the encoded guardians in the posted governance VAA. This
    /// account's guardian set index must be the next value after the current guardian set index.
    #[account(
        init,
        payer = payer,
        space = try_compute_size(&vaa)?,
        seeds = [
            GuardianSet::SEED_PREFIX,
            &curr_guardian_set.index.saturating_add(1).to_be_bytes()
        ],
        bump,
    )]
    new_guardian_set: Account<'info, LegacyAnchorized<0, GuardianSet>>,

    system_program: Program<'info, System>,
}

impl<'info> crate::utils::cpi::CreateAccount<'info> for GuardianSetUpdate<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for GuardianSetUpdate<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyGuardianSetUpdate";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = guardian_set_update;
}

impl<'info> GuardianSetUpdate<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let config = &ctx.accounts.config;
        let vaa = VaaAccount::load(&ctx.accounts.vaa)?;
        let gov_payload = super::require_valid_governance_vaa(config, &vaa)?;

        // Encoded guardian set must be the next value after the current guardian set index.
        //
        // NOTE: Because try_compute_size already determined whether this governance payload is a
        // guardian set update, we are safe to unwrap here.
        require_eq!(
            gov_payload.guardian_set_update().unwrap().new_index(),
            config.guardian_set_index + 1,
            CoreBridgeError::InvalidGuardianSetIndex
        );

        // Done.
        Ok(())
    }
}

/// Processor for guardian set update governance decrees. This instruction handler updates the
/// guardian set index in the Core Bridge [Config] account and creates a new [GuardianSet] account
/// with the new guardians encoded in the governance VAA.
#[access_control(GuardianSetUpdate::constraints(&ctx))]
fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    let vaa = VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    crate::utils::vaa::claim_vaa(ctx.accounts, &ctx.accounts.claim, &crate::ID, &vaa)?;

    let gov_payload = CoreBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .unwrap()
        .decree();
    let decree = gov_payload.guardian_set_update().unwrap();

    // Deserialize new guardian set.
    let keys: Vec<[u8; 20]> = (0..usize::from(decree.num_guardians()))
        .map(|i| decree.guardian_at(i))
        .collect();
    // We need at least one guardian for the initial guardian set.
    require!(!keys.is_empty(), CoreBridgeError::ZeroGuardians);

    for (i, guardian) in keys.iter().take(keys.len() - 1).enumerate() {
        // We disallow guardian pubkeys that have zero address.
        require!(*guardian != [0; 20], CoreBridgeError::GuardianZeroAddress);

        // Check if this pubkey is a duplicate of any others.
        for other in keys.iter().skip(i + 1) {
            require!(guardian != other, CoreBridgeError::DuplicateGuardianAddress);
        }
    }

    // We need to check the last guardian to see if it is the zero address (this guardian was missed
    // in the loop above).
    require!(
        *keys.last().unwrap() != [0; 20],
        CoreBridgeError::GuardianZeroAddress
    );

    // Set new guardian set account fields.
    let creation_time = match &vaa {
        VaaAccount::EncodedVaa(inner) => inner
            .as_vaa()
            .unwrap()
            .v1()
            .unwrap()
            .body()
            .timestamp()
            .into(),
        VaaAccount::PostedVaaV1(inner) => inner.timestamp(),
    };

    ctx.accounts.new_guardian_set.set_inner(
        GuardianSet {
            index: ctx.accounts.curr_guardian_set.index + 1,
            creation_time,
            keys,
            expiration_time: Default::default(),
        }
        .into(),
    );

    // Set the new index on the config program data.
    let config = &mut ctx.accounts.config;
    config.guardian_set_index += 1;

    // Now set the expiration time for the current guardian.
    let now = Clock::get().map(Timestamp::from)?;
    ctx.accounts.curr_guardian_set.expiration_time = now.saturating_add(&config.guardian_set_ttl);

    // Done.
    Ok(())
}

/// Read account info data assuming we are reading a guardian set update governance decree. The
/// governance decree determines the size of the guardian set account based on how many guardian
/// pubkeys are encoded in the governance VAA.
///
/// NOTE: We check the validity of the governance VAA in access control. If the posted VAA happens
/// to deserialize as a guardian set update decree but anything else is invalid about this message,
/// this instruction handler will revert (just not at this step when determining the account size).
fn try_compute_size(vaa: &AccountInfo) -> Result<usize> {
    let vaa = VaaAccount::load(vaa)?;
    let gov_payload = CoreBridgeGovPayload::try_from(vaa.try_payload()?)
        .map(|msg| msg.decree())
        .map_err(|_| error!(CoreBridgeError::InvalidGovernanceVaa))?;

    gov_payload
        .guardian_set_update()
        .map(|decree| GuardianSet::compute_size(decree.num_guardians().into()))
        .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))
}
