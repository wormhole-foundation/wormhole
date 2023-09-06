use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAnchorized,
    state::{GuardianSet, Header, ProcessingStatus},
    types::VaaVersion,
    zero_copy::EncodedVaa,
};
use anchor_lang::prelude::*;
use solana_program::{keccak, secp256k1_recover::secp256k1_recover};
use wormhole_raw_vaas::GuardianSetSig;

#[derive(Accounts)]
pub struct ProcessEncodedVaa<'info> {
    /// This account is only required to be mutable for the `CloseVaaAccount` directive. This
    /// authority is the same signer that originally created the VAA accounts, so he is the one that
    /// will receive the lamports back for the closed accounts.
    #[account(mut)]
    write_authority: Signer<'info>,

    /// CHECK: We do not deserialize this account as `VaaV1` because allocating heap memory in its
    /// deserialization uses significant compute units with every call to this instruction handler.
    /// For large VAAs, this can be a significant cost.
    ///
    /// This instruction handler performs the same checks Anchor performs:
    /// - Discriminator check (found in `AccountDeserialize`).
    /// - Write authority check (via `has_one`).
    #[account(
        mut,
        owner = crate::ID
    )]
    encoded_vaa: AccountInfo<'info>,

    /// The guardian set account is optional because it is only needed for the signature verification
    /// instruction handler directive.
    ///
    /// NOTE: Because the vaa account is not deserialized as an Anchor Account, we cannot use the
    /// guardian_set_index in `VaaV1` here easily. Instead we check that the guardian set index
    /// matches once the VAA is verified via the `VerifySignaturesV1` directive.
    #[account(
        seeds = [GuardianSet::SEED_PREFIX, &guardian_set.index.to_be_bytes()],
        bump,
    )]
    guardian_set: Option<Account<'info, LegacyAnchorized<0, GuardianSet>>>,
}

impl<'info> ProcessEncodedVaa<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        if let Some(guardian_set) = &ctx.accounts.guardian_set {
            // Guardian set must be active.
            let timestamp = Clock::get().map(Into::into)?;
            require!(
                guardian_set.is_active(&timestamp),
                CoreBridgeError::GuardianSetExpired
            );
        }

        // Check write authority.
        let acc_data = ctx.accounts.encoded_vaa.try_borrow_data()?;
        let vaa = EncodedVaa::parse_unverified(&acc_data)?;
        require_keys_eq!(
            ctx.accounts.write_authority.key(),
            vaa.write_authority(),
            CoreBridgeError::WriteAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum ProcessEncodedVaaDirective {
    /// Close the VAA processing accounts. Either someone decides to close the VAA account before it
    /// has been verified, or the VAA has been verified by an integrating app and is no longer
    /// needed.
    CloseVaaAccount,
    /// Write input data from VAA, indicated by the index of the encoded VAA.
    Write {
        /// Index of encoded VAA.
        index: u32,
        /// Data representing the encoded VAA starting at specified index.
        data: Vec<u8>,
    },
    /// When the whole VAA is written to the vaa account, its message hash is computed and guardian
    /// signatures are verified. Invoking this directive will mark the VAA as verified.
    ///
    /// NOTE: The guardian set is a required account in order to perform this method.
    VerifySignaturesV1,
}

#[access_control(ProcessEncodedVaa::constraints(&ctx))]
pub fn process_encoded_vaa(
    ctx: Context<ProcessEncodedVaa>,
    directive: ProcessEncodedVaaDirective,
) -> Result<()> {
    match directive {
        ProcessEncodedVaaDirective::CloseVaaAccount => close_vaa_account(ctx),
        ProcessEncodedVaaDirective::Write { index, data } => write(ctx, index, data),
        ProcessEncodedVaaDirective::VerifySignaturesV1 => verify_signatures_v1(ctx),
    }
}

fn close_vaa_account(ctx: Context<ProcessEncodedVaa>) -> Result<()> {
    msg!("Directive: CloseVaaAccount");

    crate::utils::close_account(
        ctx.accounts.encoded_vaa.to_account_info(),
        ctx.accounts.write_authority.to_account_info(),
    )
}

fn write(ctx: Context<ProcessEncodedVaa>, index: u32, data: Vec<u8>) -> Result<()> {
    require!(
        !data.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let vaa_size: usize = {
        let acc_data = ctx.accounts.encoded_vaa.data.borrow();
        let vaa = EncodedVaa::parse_unverified(&acc_data)?;
        require!(
            vaa.status() == ProcessingStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        vaa.vaa_size()
    };

    let index = usize::try_from(index).unwrap();
    let end = index.saturating_add(data.len());
    require_gte!(vaa_size, end, CoreBridgeError::DataOverflow);

    const START: usize = 8 + EncodedVaa::VAA_START;
    let acc_data: &mut [u8] = &mut ctx.accounts.encoded_vaa.data.borrow_mut();
    acc_data[(START + index)..(START + end)].copy_from_slice(&data);

    // Done.
    Ok(())
}

fn verify_signatures_v1(ctx: Context<ProcessEncodedVaa>) -> Result<()> {
    msg!("Directive: VerifySignaturesV1");

    require!(
        ctx.accounts.guardian_set.is_some(),
        ErrorCode::AccountNotEnoughKeys
    );

    let guardian_set = ctx.accounts.guardian_set.as_ref().unwrap();

    let write_authority = {
        let acc_data = ctx.accounts.encoded_vaa.data.borrow();
        let encoded_vaa = EncodedVaa::parse_unverified(&acc_data)?;
        require!(
            encoded_vaa.status() == ProcessingStatus::Writing,
            CoreBridgeError::VaaAlreadyVerified
        );

        // Parse and verify.
        let vaa = encoded_vaa.v1_unverified()?;

        // Must be V1.
        require_eq!(
            vaa.version(),
            u8::from(VaaVersion::V1),
            CoreBridgeError::InvalidVaaVersion
        );

        // Make sure the encoded guardian set index agrees with the guardian set account's index.
        require_eq!(
            vaa.guardian_set_index(),
            guardian_set.index,
            CoreBridgeError::GuardianSetMismatch
        );

        // Do we have enough signatures for quorum?
        let guardian_keys = &guardian_set.keys;
        let quorum = crate::utils::quorum(guardian_keys.len());
        require_gte!(
            usize::from(vaa.signature_count()),
            quorum,
            CoreBridgeError::NoQuorum
        );

        // Generate the same message hash (using keccak) that the Guardians used to generate their
        // signatures. This message hash will be hashed again to produce the digest for
        // `secp256k1_recover`.
        let digest = keccak::hash(keccak::hash(vaa.body().as_ref()).as_ref());

        // Only verify as many as we need (up to quorum).
        let mut last_guardian_index = None;
        let mut num_verified = 0;
        for sig in vaa.signatures() {
            // We do not allow for non-increasing guardian signature indices.
            let index = usize::from(sig.guardian_index());
            if let Some(last_index) = last_guardian_index {
                require_gt!(index, last_index, CoreBridgeError::InvalidGuardianIndex);
            }

            // Does this guardian index exist in this guardian set?
            let guardian_pubkey = guardian_keys
                .get(index)
                .ok_or_else(|| error!(CoreBridgeError::InvalidGuardianIndex))?;

            // Now verify that the signature agrees with the expected Guardian's pubkey.
            verify_guardian_signature(&sig, guardian_pubkey, digest.as_ref())?;
            num_verified += 1;

            // If we have reached quorum, no need to spend compute units to verify other signatures.
            if num_verified == quorum {
                break;
            }

            last_guardian_index = Some(index);
        }

        encoded_vaa.write_authority()
    };

    // Skip discriminator.
    let acc_data: &mut [u8] = &mut ctx.accounts.encoded_vaa.data.borrow_mut()[8..];
    let mut writer = std::io::Cursor::new(acc_data);
    Header {
        status: ProcessingStatus::Verified,
        write_authority,
        version: VaaVersion::V1,
    }
    .serialize(&mut writer)
    .map_err(Into::into)
}

fn verify_guardian_signature(
    sig: &GuardianSetSig,
    guardian_pubkey: &[u8; 20],
    digest: &[u8],
) -> Result<()> {
    // Recover using `solana_program::secp256k1_recover`. Public key recovery costs 25k compute
    // units. And hashing this public key to recover the Ethereum public key costs about 13k.
    let recovered = {
        // Recover EC public key (64 bytes).
        let pubkey = secp256k1_recover(digest, sig.recovery_id(), &sig.rs())
            .map_err(|_| CoreBridgeError::InvalidSignature)?;

        // The Ethereum public key is the last 20 bytes of keccak hashed public key above.
        let hashed = keccak::hash(&pubkey.to_bytes());

        let mut eth_pubkey = [0; 20];
        eth_pubkey.copy_from_slice(&hashed.0[12..]);

        eth_pubkey
    };

    // The recovered public key should agree with the Guardian's public key at this index.
    require!(
        recovered == *guardian_pubkey,
        CoreBridgeError::InvalidGuardianKeyRecovery
    );

    // Done.
    Ok(())
}
