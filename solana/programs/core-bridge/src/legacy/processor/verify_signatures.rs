use crate::{
    error::CoreBridgeError,
    legacy::{instruction::VerifySignaturesArgs, utils::LegacyAnchorized},
    state::{GuardianSet, SignatureSet},
    types::MessageHash,
};
use anchor_lang::{prelude::*, solana_program::sysvar};

/// Offset schema used by the Sig Verify native program.
#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, InitSpace)]
struct SigVerifyOffsets {
    /// Offset to \[signature,recovery_id,etherum_address\] of 64 + 1 + 20 bytes.
    signature_offset: u16,
    /// Instruction index to find signature data.
    signature_ix_index: u8,
    /// Offset to \[signature,recovery_id\] of 64 + 1 bytes.
    eth_pubkey_offset: u16,
    // Instruction index to find eth pubkey data.
    eth_pubkey_ix_index: u8,
    // Offset to start of message data.
    message_offset: u16,
    // Size of message data.
    message_size: u16,
    // Index of instruction data to get message data.
    message_ix_index: u8,
}

/// Result of parsing Sig Verify instruction data.
struct SigVerifyParameters {
    eth_pubkeys: Vec<[u8; 20]>,
    message: [u8; 32],
}

#[derive(Accounts)]
pub struct VerifySignatures<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// Guardian set used for signature verification. These pubkeys were passed into the Sig Verify
    /// native program to do its signature verification.
    #[account(
        seeds = [GuardianSet::SEED_PREFIX, &guardian_set.index.to_be_bytes()],
        bump,
    )]
    guardian_set: Account<'info, LegacyAnchorized<0, GuardianSet>>,

    /// Stores signature validation from Sig Verify native program.
    #[account(
        init_if_needed,
        payer = payer,
        space = SignatureSet::compute_size(guardian_set.keys.len())
    )]
    signature_set: Account<'info, LegacyAnchorized<0, SignatureSet>>,

    /// CHECK: Instruction sysvar used to read Sig Verify native program instruction data.
    #[account(
        address = sysvar::instructions::id() @ ErrorCode::AccountSysvarMismatch
    )]
    instructions: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, VerifySignaturesArgs>
    for VerifySignatures<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyVerifySignatures";

    const ANCHOR_IX_FN: fn(Context<Self>, VerifySignaturesArgs) -> Result<()> = verify_signatures;
}

impl<'info> VerifySignatures<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Check that the guardian set is still active.
        //
        // NOTE: The legacy implementation was not able to perform this check on the guardian
        // set. We can now short-circuit VAA verification failure by checking whether a guardian
        // set is expired here instead of having to wait for the invoking the post VAA instruction
        // handler.
        let timestamp = Clock::get().map(Into::into)?;
        require!(
            ctx.accounts.guardian_set.is_active(&timestamp),
            CoreBridgeError::PostVaaGuardianSetExpired
        );

        // Done.
        Ok(())
    }
}

/// Processor to verify guardian signatures leveraging the Sig Verify native program to perform the
/// elliptic curve signature verification. This instruction is scary because it relies on checking
/// Sig Verify instruction data to make sure that the correct signatures are being verified.
///
/// NOTE: It is recommended that VAAs be verified using the new Anchor instructions
/// `init_encoded_vaa` and `process_encoded_vaa`, which does not rely on the Sig Verify native
/// program to verify elliptic curve signatures. Also, using this instruction is inefficient because
/// it requires additional data encoded in a transaction: the Sig Verify native program requires
/// guardian pubkeys to be provided in order to perform its signature verification.
#[access_control(VerifySignatures::constraints(&ctx))]
fn verify_signatures(ctx: Context<VerifySignatures>, args: VerifySignaturesArgs) -> Result<()> {
    let VerifySignaturesArgs { signer_indices } = args;

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

    // It would have been nice to be able to perform this check in `access_control`, but there
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

    // And here we verify that the previous instruction is actually the Sig Verify native program.
    //
    // NOTE: To avoid a redundant instructions sysvar check, we allow the deprecated method to
    // load the instruction data.
    #[allow(deprecated)]
    let SigVerifyParameters {
        eth_pubkeys: signers,
        message,
    } = sysvar::instructions::load_instruction_at(
        usize::from(sig_verify_index),
        &instruction_sysvar_data,
    )
    .map_err(|_| ProgramError::InvalidInstructionData.into())
    .and_then(|ix| deserialize_secp256k1_ix(sig_verify_index, &ix))?;

    // Number of specified signers must equal the number of signatures verified in the Sig Verify
    // native program instruction.
    require_eq!(
        signers.len(),
        guardian_indices.len(),
        CoreBridgeError::SignerIndicesMismatch
    );

    // We use this message hash later on.
    let message_hash = MessageHash::from(message);
    let signature_set = &mut ctx.accounts.signature_set;

    // If the signature set account has not been initialized yet, establish the expected account
    // data (guardian set index used, hash and which indices have been validated).
    if signature_set.is_initialized() {
        // Otherwise, verify that the guardian set index is what we expect from
        // the last time we wrote to the signature set account.
        require_eq!(
            guardian_set.index,
            signature_set.guardian_set_index,
            CoreBridgeError::GuardianSetMismatch
        );

        // And verify that the message hash is the same as the one already encoded in the signature
        // set.
        require_eq!(
            message_hash,
            signature_set.message_hash,
            CoreBridgeError::MessageMismatch
        );
    } else {
        // We're assuming that the hashed Wormhole message is not zero bytes.
        // So if the account data is all zeros, we're assuming that the account
        // is created at this instruction call. Save the guardian set index and
        // message hash.
        signature_set.set_inner(
            SignatureSet {
                sig_verify_successes: vec![false; guardians.len()],
                message_hash,
                guardian_set_index: guardian_set.index,
            }
            .into(),
        );
    }

    // Attempt to write `true` to represent verified guardian eth pubkey.
    for (i, &signer_index) in guardian_indices.iter().enumerate() {
        require!(
            signers[i] == guardians[signer_index],
            CoreBridgeError::InvalidGuardianKeyRecovery
        );

        // Overwritten content should be zeros except double signs by the
        // signer or harmless replays.
        signature_set.sig_verify_successes[signer_index] = true;
    }

    // Done.
    Ok(())
}

/// This method performs the Sig Verify native program instruction deserialization and validates
/// this data.
fn deserialize_secp256k1_ix(
    sig_verify_index: u16,
    ix: &solana_program::instruction::Instruction,
) -> Result<SigVerifyParameters> {
    // Check that the program invoked is the secp256k1 program.
    require_keys_eq!(
        ix.program_id,
        solana_program::secp256k1_program::id(),
        CoreBridgeError::InvalidSigVerifyInstruction
    );

    let ix_data = &ix.data;

    // The first byte encodes the number of signatures.
    let num_signatures: usize = ix_data[0].into();

    let mut eth_pubkeys = Vec::with_capacity(num_signatures);

    // For each offset encoded, grab each SigVerify parameter (signature, eth pubkey, message).
    let mut expected_message_offset = None;
    for i in 0..num_signatures {
        let offsets_idx = 1 + i * SigVerifyOffsets::INIT_SPACE;
        let SigVerifyOffsets {
            signature_offset: _,
            signature_ix_index,
            eth_pubkey_offset,
            eth_pubkey_ix_index,
            message_offset,
            message_size,
            message_ix_index,
        } = SigVerifyOffsets::deserialize(
            &mut &ix_data[offsets_idx..(offsets_idx + SigVerifyOffsets::INIT_SPACE)],
        )?;
        // Because guardians sign the hash of the message body hash, this verified message must be
        // 32 bytes.
        require_eq!(
            message_size,
            32,
            CoreBridgeError::InvalidSigVerifyInstruction
        );

        // The instruction index must be the same for signature, eth pubkey and message.
        require_eq!(
            u16::from(signature_ix_index),
            sig_verify_index,
            CoreBridgeError::InvalidSigVerifyInstruction
        );
        require_eq!(
            u16::from(eth_pubkey_ix_index),
            sig_verify_index,
            CoreBridgeError::InvalidSigVerifyInstruction
        );
        require_eq!(
            u16::from(message_ix_index),
            sig_verify_index,
            CoreBridgeError::InvalidSigVerifyInstruction
        );

        let eth_pubkey_offset = usize::from(eth_pubkey_offset);
        let mut eth_pubkey = [0; 20];
        eth_pubkey.copy_from_slice(&ix_data[eth_pubkey_offset..(eth_pubkey_offset + 20)]);

        // The message offset should be the same for each sig verify offsets since each signature is
        // for the same message.
        let message_offset = usize::from(message_offset);
        if let Some(expected_message_offset) = expected_message_offset {
            require_eq!(
                message_offset,
                expected_message_offset,
                CoreBridgeError::InvalidSigVerifyInstruction
            );
        }

        eth_pubkeys.push(eth_pubkey);
        expected_message_offset = Some(message_offset);
    }

    if let Some(message_offset) = expected_message_offset {
        let mut message = [0; 32];
        message.copy_from_slice(&ix_data[message_offset..(message_offset + 32)]);

        Ok(SigVerifyParameters {
            eth_pubkeys,
            message,
        })
    } else {
        Err(CoreBridgeError::EmptySigVerifyInstruction.into())
    }
}
