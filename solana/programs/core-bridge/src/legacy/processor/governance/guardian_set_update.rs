use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, Config, GuardianSet, PartialPostedVaaV1, VaaV1Account},
    types::Timestamp,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{utils, NewAccountSize, SeedPrefix};

#[derive(Accounts)]
pub struct GuardianSetUpdate<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, Config>,

    #[account(
        seeds = [
            PartialPostedVaaV1::SEED_PREFIX,
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PartialPostedVaaV1>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes()
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    #[account(
        mut,
        seeds = [GuardianSet::SEED_PREFIX, &config.guardian_set_index.to_be_bytes()],
        bump,
    )]
    curr_guardian_set: Account<'info, GuardianSet>,

    #[account(
        init,
        payer = payer,
        space = try_compute_size(&posted_vaa)?,
        seeds = [GuardianSet::SEED_PREFIX, &(curr_guardian_set.index + 1).to_be_bytes()],
        bump,
    )]
    new_guardian_set: Account<'info, GuardianSet>,

    system_program: Program<'info, System>,
}

/// Read account info data assuming we are reading a guardian set update governance decree to
/// determine how large the new guardian set account needs to be given how many guardians there are
/// in the new set.
///
/// NOTE: This step does not have to fail because if this VAA is not what we expect, the access
/// control following the account context checks will fail.
fn try_compute_size(posted_vaa: &Account<PartialPostedVaaV1>) -> Result<usize> {
    let acc_info: &AccountInfo = posted_vaa.as_ref();
    let acc_data = acc_info.try_borrow_data()?;

    let gov_payload = super::parse_gov_payload(&acc_data)?;

    match gov_payload.decree().guardian_set_update() {
        Some(decree) => Ok(GuardianSet::compute_size(decree.num_guardians().into())),
        None => err!(CoreBridgeError::InvalidGovernanceAction),
    }
}

impl<'info> GuardianSetUpdate<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let config = &ctx.accounts.config;
        let vaa = &ctx.accounts.posted_vaa;

        let acc_info: &AccountInfo = vaa.as_ref();
        let acc_data = acc_info.data.borrow();

        let gov_payload = super::require_valid_governance_posted_vaa(
            vaa.details(),
            &acc_data,
            vaa.guardian_set_index,
            config,
        )?;

        // Encoded guardian set must be the next value after the current guardian set index.
        //
        // NOTE: Because try_compute_size already determined whether this governance payload is a
        // guardian set update, we are safe to unwrap here.
        require_eq!(
            gov_payload
                .decree()
                .guardian_set_update()
                .unwrap()
                .new_index(),
            config.guardian_set_index + 1,
            CoreBridgeError::InvalidGuardianSetIndex
        );

        // Done.
        Ok(())
    }
}

#[access_control(GuardianSetUpdate::constraints(&ctx))]
pub fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let acc_data = acc_info.data.borrow();

    let gov_payload = super::parse_gov_payload(&acc_data).unwrap().decree();
    let decree = gov_payload.guardian_set_update().unwrap();

    // Deserialize new guardian set.
    let keys: Vec<[u8; 20]> = (0..usize::from(decree.num_guardians()))
        .map(|i| decree.guardian_at(i))
        .collect();
    for (i, guardian) in keys.iter().take(keys.len() - 1).enumerate() {
        // We disallow guardian pubkeys that have zero address.
        require!(
            utils::is_nonzero_array(guardian),
            CoreBridgeError::GuardianZeroAddress
        );

        // Check if this pubkey is a duplicate of any others.
        for other in keys.iter().skip(i + 1) {
            require!(guardian != other, CoreBridgeError::DuplicateGuardianAddress);
        }
    }

    // Set new guardian set account fields.
    ctx.accounts.new_guardian_set.set_inner(GuardianSet {
        index: ctx.accounts.curr_guardian_set.index + 1,
        creation_time: ctx.accounts.posted_vaa.timestamp,
        keys,
        expiration_time: Default::default(),
    });

    // Set the new index on the config program data.
    let config = &mut ctx.accounts.config;
    config.guardian_set_index += 1;

    // Now set the expiration time for the current guardian.
    let now = Clock::get().map(Timestamp::from)?;
    ctx.accounts.curr_guardian_set.expiration_time = now.saturating_add(&config.guardian_set_ttl);

    // Done.
    Ok(())
}
