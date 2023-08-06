use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{BridgeProgramData, Claim, GuardianSet, PartialPostedVaaV1, VaaV1MessageHash},
    types::Timestamp,
    utils::GOVERNANCE_DECREE_START,
};
use anchor_lang::prelude::*;
use wormhole_io::Readable;
use wormhole_solana_common::{utils, NewAccountSize, SeedPrefix};

const ACTION_GUARDIAN_SET_UPDATE: u8 = 2;

#[derive(Accounts)]
pub struct GuardianSetUpdate<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
    )]
    bridge: Account<'info, BridgeProgramData>,

    #[account(
        seeds = [
            PartialPostedVaaV1::seed_prefix(),
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
        seeds = [GuardianSet::seed_prefix(), &bridge.guardian_set_index.to_be_bytes()],
        bump,
    )]
    curr_guardian_set: Account<'info, GuardianSet>,

    #[account(
        init,
        payer = payer,
        space = try_compute_size(&posted_vaa)?,
        seeds = [GuardianSet::seed_prefix(), &(curr_guardian_set.index + 1).to_be_bytes()],
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

    // Start slice where the encoded number of guardians is.
    let num_guardians = acc_info
        .try_borrow_data()
        .map(|data| data[GOVERNANCE_DECREE_START + 4])
        .map_err(|_| ErrorCode::AccountDidNotDeserialize)?;

    Ok(GuardianSet::compute_size(num_guardians.into()))
}

impl<'info> GuardianSetUpdate<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let bridge = &ctx.accounts.bridge;
        let action =
            crate::utils::require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa, bridge)?;

        require_eq!(
            action,
            ACTION_GUARDIAN_SET_UPDATE,
            CoreBridgeError::InvalidGovernanceAction
        );

        let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
        let mut data = &acc_info.data.borrow()[GOVERNANCE_DECREE_START..];

        // Encoded guardian set must be the next value after the current guardian set index.
        let new_index = u32::read(&mut data).unwrap();
        require_eq!(new_index, bridge.guardian_set_index + 1);

        // Done.
        Ok(())
    }
}

#[access_control(GuardianSetUpdate::accounts(&ctx))]
pub fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // We have already checked that the new guardian set index is correct, so we can just add one to
    // the existing index and begin deserializing the guardian set.
    let new_index = ctx.accounts.curr_guardian_set.index + 1;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let mut data = &acc_info.data.borrow()[(GOVERNANCE_DECREE_START + 4)..];

    // Deserialize new guardian set.
    let num_guardians = u8::read(&mut data).unwrap();
    let keys: Vec<[u8; 20]> = (0..num_guardians)
        .map(|_| Readable::read(&mut data).unwrap())
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

    let now = Clock::get().map(Timestamp::from)?;

    // Set new guardian set account fields.
    ctx.accounts.new_guardian_set.set_inner(GuardianSet {
        index: new_index,
        creation_time: now,
        keys,
        expiration_time: Default::default(),
    });

    // Set the new index on the bridge program data.
    let bridge = &mut ctx.accounts.bridge;
    bridge.guardian_set_index = new_index;

    // Now set the expiration time for the current guardian.
    ctx.accounts.curr_guardian_set.expiration_time = now.saturating_add(&bridge.guardian_set_ttl);

    // Done.
    Ok(())
}
