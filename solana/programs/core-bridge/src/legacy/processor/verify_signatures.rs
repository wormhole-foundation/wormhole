use crate::{
    error::CoreBridgeError,
    legacy::instruction::LegacyVerifySignaturesArgs,
    state::{GuardianSet, SignatureSet},
    types::MessageHash,
};
use anchor_lang::{
    error,
    prelude::*,
    solana_program::{keccak, sysvar},
};
use wormhole_solana_common::{NewAccountSize, SeedPrefix};

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy)]
pub struct SigVerifyOffsets {
    pub signature_offset: u16, // offset to [signature,recovery_id,etherum_address] of 64+1+20 bytes
    pub signature_ix_index: u8, // instruction index to find data
    pub eth_pubkey_offset: u16, // offset to [signature,recovery_id] of 64+1 bytes
    pub eth_pubkey_ix_index: u8, // instruction index to find data
    pub message_offset: u16,   // offset to start of message data
    pub message_size: u16,     // size of message data
    pub message_ix_index: u8,  // index of instruction data to get message data
}

impl SigVerifyOffsets {
    pub const LEN: usize = 2    // signature_key_offset
        + 1                     // signature_instruction_index
        + 2                     // pubkey_offset
        + 1                     // pubkey_instruction_index
        + 2                     // message_data_offset
        + 2                     // message_data_size
        + 1                     // message_instruction_index
    ;
}

struct SigVerifyParameters {
    eth_pubkey: [u8; 20],
    message: [u8; 32],
}

#[derive(Accounts)]
pub struct VerifySignatures<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// Guardian set used for signature verification. The legacy instruction does not check if the
    /// guardian set is allowed or has not expired yet.
    ///
    /// Currently signatures from a past guardian set can be verified, which is a waste of compute
    /// units since the post_vaa instruction will fail if the guardian set is not active.
    #[account(
        seeds = [GuardianSet::SEED_PREFIX, &guardian_set.index.to_be_bytes()],
        bump,
    )]
    guardian_set: Account<'info, GuardianSet>,

    /// Stores signature validation from libsecp256k1 program.
    #[account(
        init_if_needed,
        payer = payer,
        space = SignatureSet::compute_size(guardian_set.keys.len())
    )]
    signature_set: Account<'info, SignatureSet>,

    /// CHECK: Instruction sysvar used to read libsecp256k1 instruction data.
    #[account(
        address = sysvar::instructions::id() @ error::ErrorCode::AccountSysvarMismatch
    )]
    instructions: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> VerifySignatures<'info> {
    /// This method performs an additional context constraint where we use the Clock sysvar to
    /// determine whether the guardian set is still active.
    ///
    /// NOTE: The previous implementation required the Clock sysvar to be defined as a part of the
    /// accounts context in order to perform this check. By performing this check here, we can fail
    /// earlier (as opposed to failing at the `post_vaa` step after verifying all the signatures
    /// with a potentially expired guardian set).
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let timestamp = Clock::get().map(Into::into)?;
        let guardian_set = &ctx.accounts.guardian_set;
        require!(
            guardian_set.is_active(&timestamp),
            CoreBridgeError::PostVaaGuardianSetExpired
        );

        Ok(())
    }
}

#[access_control(VerifySignatures::constraints(&ctx))]
pub fn verify_signatures(
    ctx: Context<VerifySignatures>,
    args: LegacyVerifySignaturesArgs,
) -> Result<()> {
    let LegacyVerifySignaturesArgs { signer_indices } = args;

    // Collected guardian indices (to be used later).
    let guardian_indices: Vec<_> = signer_indices
        .iter()
        .enumerate()
        .filter_map(|(i, &value)| if value >= 0 { Some(i) } else { None })
        .collect();

    // Before we continue, check that the array argument passed into this instruction is valid.
    let guardian_set = &ctx.accounts.guardian_set;
    let guardians = &guardian_set.keys;
    require!(
        !guardian_indices.is_empty() && *guardian_indices.last().unwrap() < guardians.len(),
        CoreBridgeError::InvalidInstructionArgument
    );

    // NOTE: It would have been nice to be able to perform this check in `access_control`, but there
    // is no data from the instruction sysvar loaded by that point. We have to load it and perform
    // the safety checks in this instruction handler.
    let instruction_sysvar_data = ctx.accounts.instructions.data.borrow();

    // We grab the index of the instruction before this instruction, which should be the sig verify
    // program. To avoid a redundant Instructions sysvar check, we allow this deprecated method.
    //
    // NOTE: To avoid a redundant instructions sysvar check, we allow the deprecated method to
    // load the instruction data.
    #[allow(deprecated)]
    let sig_verify_index = u16::checked_sub(
        sysvar::instructions::load_current_index(&instruction_sysvar_data),
        1,
    )
    .ok_or(CoreBridgeError::InstructionAtWrongIndex)?;

    // And here we verify that the previous instruction is actually the `secp256k1_program`. To
    // avoid a redundant Instructions sysvar check, we allow this deprecated method.
    #[allow(deprecated)]
    let sig_verify_params = sysvar::instructions::load_instruction_at(
        usize::from(sig_verify_index),
        &instruction_sysvar_data,
    )
    .map_err(|_| ProgramError::InvalidInstructionData.into())
    .and_then(|ix| deserialize_secp256k1_ix(&ix))?;

    // Number of specified `signers` must equal the number of signatures verified in the sig verify
    // program instruction.
    require_eq!(
        guardian_indices.len(),
        sig_verify_params.len(),
        CoreBridgeError::SignerIndicesMismatch
    );

    // We're going to use this message data later on.
    let message_hash = MessageHash::from(sig_verify_params[0].message);
    let signature_set = &mut ctx.accounts.signature_set;

    // If the signature set account has not been initialized yet, establish the expected account
    // data (guardian set index used, hash and which indices have been validated).
    if signature_set.is_initialized() {
        // Otherwise, verify that the guardian set index is what we expect from
        // the last time we wrote to the signature set account.
        require_eq!(
            signature_set.guardian_set_index,
            guardian_set.index,
            CoreBridgeError::GuardianSetMismatch
        );

        // And verify that the message hash is the same.
        require!(
            signature_set.message_hash == message_hash,
            CoreBridgeError::MessageMismatch
        );
    } else {
        // We're assuming that the hashed Wormhole message is not zero bytes.
        // So if the account data is all zeros, we're assuming that the account
        // is created at this instruction call. Save the guardian set index and
        // message hash.
        signature_set.set_inner(SignatureSet {
            sig_verify_successes: vec![false; guardians.len()],
            message_hash,
            guardian_set_index: guardian_set.index,
        });
    }

    // Attempt to write `true` to represent verified guardian eth pubkey.
    for (i, &signer_index) in guardian_indices.iter().enumerate() {
        require!(
            sig_verify_params[i].eth_pubkey == guardians[signer_index],
            CoreBridgeError::InvalidGuardianKeyRecovery
        );

        // Overwritten content should be zeros except double signs by the
        // signer or harmless replays.
        signature_set.sig_verify_successes[signer_index] = true;
    }

    // Done.
    Ok(())
}

fn deserialize_secp256k1_ix(
    ix: &solana_program::instruction::Instruction,
) -> Result<Vec<SigVerifyParameters>> {
    // Check that the program invoked is the secp256k1 program.
    require_keys_eq!(
        ix.program_id,
        solana_program::secp256k1_program::id(),
        CoreBridgeError::InvalidSigVerifyInstruction
    );

    let ix_data = &ix.data;

    // First byte encodes the number of signatures.
    let mut params = Vec::with_capacity(ix_data[0].into());

    // For each offset encoded, grab each SigVerify parameter (signature, eth pubkey, message).
    let mut last_message_offset = None;
    for i in 0..params.capacity() {
        let offsets_idx = 1 + i * SigVerifyOffsets::LEN;
        let offsets = SigVerifyOffsets::deserialize(
            &mut &ix_data[offsets_idx..(offsets_idx + SigVerifyOffsets::LEN)],
        )?;
        // Because guardians sign the hash of the message body hash, this verified message must be
        // 32 bytes.
        require_eq!(
            offsets.message_size,
            32,
            CoreBridgeError::InvalidSigVerifyInstruction
        );

        // The instruction index must be the same for signature, eth pubkey and message.
        require_eq!(
            offsets.signature_ix_index,
            offsets.eth_pubkey_ix_index,
            CoreBridgeError::InvalidSigVerifyInstruction
        );
        require_eq!(
            offsets.signature_ix_index,
            offsets.message_ix_index,
            CoreBridgeError::InvalidSigVerifyInstruction
        );

        let eth_pubkey_offset = usize::from(offsets.eth_pubkey_offset);
        let mut eth_pubkey = [0; 20];
        eth_pubkey.copy_from_slice(&ix_data[eth_pubkey_offset..(eth_pubkey_offset + 20)]);

        // The message offset should be the same for each sig verify offsets since each signature is
        // for the same message.
        let message_offset = usize::from(offsets.message_offset);
        if let Some(last_message_offset) = last_message_offset {
            require_eq!(
                message_offset,
                last_message_offset,
                CoreBridgeError::InvalidSigVerifyInstruction
            );
        }

        let mut message = [0; keccak::HASH_BYTES];
        message.copy_from_slice(&ix_data[message_offset..(message_offset + keccak::HASH_BYTES)]);

        params.push(SigVerifyParameters {
            eth_pubkey,
            message,
        });
        last_message_offset = Some(message_offset);
    }

    Ok(params)
}
