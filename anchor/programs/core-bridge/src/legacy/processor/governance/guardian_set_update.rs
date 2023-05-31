use std::io;

use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    message::{
        require_valid_governance_posted_vaa, CoreBridgeGovernance, PostedGovernanceVaaV1,
        WormDecode, WormEncode,
    },
    state::{BridgeProgramData, Claim, Guardian, GuardianSet, VaaV1LegacyAccount},
    types::Timestamp,
};
use anchor_lang::prelude::*;
use wormhole_common::{utils, NewAccountSize, SeedPrefix};

#[derive(Debug, Clone)]
struct GuardianSetUpdateDecree {
    new_guardian_set_index: u32,
    guardians: Vec<Guardian>,
}

impl WormDecode for GuardianSetUpdateDecree {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let new_guardian_set_index = u32::decode_reader(reader)?;
        let num_guardians = u8::decode_reader(reader)?;
        let guardians: Vec<_> = (0..num_guardians)
            .filter_map(|_| Guardian::decode_reader(reader).ok())
            .collect();
        if usize::from(num_guardians) != guardians.len() {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                "invalid guardian set update",
            ));
        }

        Ok(Self {
            new_guardian_set_index,
            guardians,
        })
    }
}

impl WormEncode for GuardianSetUpdateDecree {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.new_guardian_set_index.encode(writer)?;
        // This operation is safe because serialization/encoding only happens after the account has
        // been deserialized (when the message hash is recomputed).
        u8::try_from(self.guardians.len()).unwrap().encode(writer)?;
        for guardian in &self.guardians {
            guardian.encode(writer)?;
        }
        Ok(())
    }
}

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
            PostedGovernanceVaaV1::<GuardianSetUpdateDecree>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1<GuardianSetUpdateDecree>>,

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
        space = GuardianSet::compute_size(posted_vaa.payload.decree.guardians.len()),
        seeds = [GuardianSet::seed_prefix(), &(curr_guardian_set.index + 1).to_be_bytes()],
        bump,
    )]
    new_guardian_set: Account<'info, GuardianSet>,

    system_program: Program<'info, System>,
}

impl<'info> GuardianSetUpdate<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let bridge = &ctx.accounts.bridge;
        let msg = require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa, bridge)?;

        // We expect a specific governance header.
        require!(
            msg.header == CoreBridgeGovernance::GuardianSetUpdate.try_into()?,
            CoreBridgeError::InvalidGovernanceAction
        );

        let decree = &msg.decree;

        // Encoded guardian set must be the next value after the current guardian set index.
        require_eq!(decree.new_guardian_set_index, bridge.guardian_set_index + 1);

        let guardians = &decree.guardians;
        for (i, guardian) in guardians.iter().take(guardians.len() - 1).enumerate() {
            // We disallow guardian pubkeys that have zero address.
            require!(
                utils::is_nonzero_array(guardian.as_ref()),
                CoreBridgeError::GuardianZeroAddress
            );

            // Check if this pubkey is a duplicate of any others.
            for other in guardians.iter().skip(i + 1) {
                require!(guardian != other, CoreBridgeError::DuplicateGuardianAddress);
            }
        }

        // Done.
        Ok(())
    }
}

#[access_control(GuardianSetUpdate::accounts(&ctx))]
pub fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let decree = &mut ctx.accounts.posted_vaa.payload.decree;
    let index = decree.new_guardian_set_index;
    let keys = std::mem::take(&mut decree.guardians);

    // Set new guardian set account fields.
    ctx.accounts.new_guardian_set.set_inner(GuardianSet {
        index,
        creation_time: Clock::get().map(Into::into)?,
        keys,
        expiration_time: Default::default(),
    });

    // Set the new index on the bridge program data.
    let bridge = &mut ctx.accounts.bridge;
    bridge.guardian_set_index = index;

    // Now set the expiration time for the current guardian.
    ctx.accounts.curr_guardian_set.expiration_time = Clock::get()
        .map(|clock| Timestamp::from(clock).saturating_add(&bridge.guardian_set_ttl))?;

    // Done.
    Ok(())
}
