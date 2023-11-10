use crate::{
    constants::MAX_MESSAGE_PAYLOAD_SIZE,
    error::CoreBridgeError,
    legacy::utils::LegacyAccount,
    state::{MessageStatus, PostedMessageV1, PostedMessageV1Info},
    types::Commitment,
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct InitMessageV1<'info> {
    /// This authority is the only one who can write to the draft message account.
    emitter_authority: Signer<'info>,

    /// CHECK: This account will have been created using the system program outside of the Core
    /// Bridge.
    #[account(
        mut,
        owner = crate::ID
    )]
    draft_message: AccountInfo<'info>,
}

impl<'info> InitMessageV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Infer the expected message length given the size of the created account.
        let data_len = ctx.accounts.draft_message.data_len();
        require!(
            data_len > PostedMessageV1::PAYLOAD_START,
            CoreBridgeError::InvalidCreatedAccountSize
        );

        // This message length cannot exceed the maximum message length.
        require!(
            data_len - PostedMessageV1::PAYLOAD_START <= MAX_MESSAGE_PAYLOAD_SIZE,
            CoreBridgeError::ExceedsMaxPayloadSize
        );

        // Check that the message account header is completely zeroed out. By doing this, we make
        // the assumption that no other Core Bridge account that is currently used will have all
        // zeros. Ideally all of the Core Bridge accounts should have a discriminator so we do not
        // have to mess around like this. But here we are.
        let msg_acc_data: &[_] = &ctx.accounts.draft_message.try_borrow_data()?;
        require!(
            msg_acc_data[..PostedMessageV1::PAYLOAD_START] == [0; PostedMessageV1::PAYLOAD_START],
            CoreBridgeError::AccountNotZeroed
        );

        // Done.
        Ok(())
    }
}

/// Arguments for the [init_message_v1](crate::wormhole_core_bridge_solana::init_message_v1)
/// instruction.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitMessageV1Args {
    /// Unique id for this message.
    pub nonce: u32,
    /// Solana commitment level for Guardian observation.
    pub commitment: Commitment,
    /// Optional program ID if the emitter address will be your program ID.
    ///
    /// NOTE: If `Some(program_id)`, your emitter authority seeds to be \[b"emitter\].
    pub cpi_program_id: Option<Pubkey>,
}

#[access_control(InitMessageV1::constraints(&ctx))]
pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
    let expected_msg_length =
        ctx.accounts.draft_message.data_len() - PostedMessageV1::PAYLOAD_START;

    let InitMessageV1Args {
        nonce,
        commitment,
        cpi_program_id,
    } = args;

    // This instruction allows a program to declare its own program ID as an emitter if we can
    // derive the emitter authority from seeds [b"emitter"]. This is useful for programs that do not
    // want to manage two separate addresses (program ID and emitter address) cross chain.
    let emitter = match cpi_program_id {
        Some(program_id) => {
            let (expected_authority, _) = Pubkey::find_program_address(
                &[crate::constants::PROGRAM_EMITTER_SEED_PREFIX],
                &program_id,
            );
            require_eq!(
                ctx.accounts.emitter_authority.key(),
                expected_authority,
                CoreBridgeError::InvalidProgramEmitter
            );

            program_id
        }
        None => {
            // Make sure this emitter is not executable. This check is a security measure to prevent
            // someone impersonating his program as the emitter address if he still holds the
            // keypair used to deploy his program.
            require!(
                !ctx.accounts.emitter_authority.executable,
                CoreBridgeError::ExecutableDisallowed
            );

            ctx.accounts.emitter_authority.key()
        }
    };

    let acc_data: &mut [_] = &mut ctx.accounts.draft_message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);

    // Finally initialize the draft message account by serializing the discriminator, header and
    // payload length.
    std::io::Write::write_all(&mut writer, PostedMessageV1::DISCRIMINATOR)?;
    (
        PostedMessageV1Info {
            consistency_level: commitment.into(),
            emitter_authority: ctx.accounts.emitter_authority.key(),
            status: MessageStatus::Writing,
            _gap_0: Default::default(),
            posted_timestamp: Default::default(),
            nonce,
            sequence: Default::default(),
            solana_chain_id: Default::default(),
            emitter,
        },
        u32::try_from(expected_msg_length).unwrap(),
    )
        .serialize(&mut writer)
        .map_err(Into::into)
}
