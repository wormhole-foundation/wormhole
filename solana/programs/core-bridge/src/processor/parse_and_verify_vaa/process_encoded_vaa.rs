use crate::{
    error::CoreBridgeError,
    state::{EncodedVaa, GuardianSet, ProcessingHeader, ProcessingStatus},
    types::VaaVersion,
};
use anchor_lang::prelude::*;
use solana_program::{keccak, secp256k1_recover::secp256k1_recover};
use wormhole_raw_vaas::{GuardianSetSig, Vaa};
use wormhole_solana_common::{utils, SeedPrefix};
//use wormhole_vaas::GuardianSetSig;

const START: usize = EncodedVaa::BYTES_START;

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

    /// The guardian set account is optional because it is only needed for the signature verifcation
    /// instruction handler directive.
    ///
    /// NOTE: Because the vaa account is not deserialized as an Anchor Account, we cannot use the
    /// guardian_set_index in `VaaV1` here easily. Instead we check that the guardian set index
    /// matches once the VAA is verified via the `VerifySignaturesV1` directive.
    #[account(
        seeds = [GuardianSet::seed_prefix(), &guardian_set.index.to_be_bytes()],
        bump,
    )]
    guardian_set: Option<Account<'info, GuardianSet>>,
}

impl<'info> ProcessEncodedVaa<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        if let Some(guardian_set) = &ctx.accounts.guardian_set {
            // Guardian set must be active.
            let timestamp = Clock::get().map(Into::into)?;
            require!(
                guardian_set.is_active(&timestamp),
                CoreBridgeError::GuardianSetExpired
            );
        }

        // Check header.
        let mut data: &[u8] = &ctx.accounts.encoded_vaa.try_borrow_data()?;
        let header = ProcessingHeader::try_account_deserialize(&mut data)?;
        require_keys_eq!(header.write_authority, ctx.accounts.write_authority.key());

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

#[access_control(ProcessEncodedVaa::accounts(&ctx))]
pub fn process_encoded_vaa(
    ctx: Context<ProcessEncodedVaa>,
    directive: ProcessEncodedVaaDirective,
) -> Result<()> {
    let vaa_acc_info = &ctx.accounts.encoded_vaa;
    match directive {
        ProcessEncodedVaaDirective::CloseVaaAccount => {
            msg!("Directive: CloseVaaAccount");
            utils::close_account(
                vaa_acc_info.to_account_info(),
                ctx.accounts.write_authority.to_account_info(),
            )
        }
        ProcessEncodedVaaDirective::Write { index, data } => {
            msg!("Directive: Write");
            write_vaa(
                vaa_acc_info,
                index
                    .try_into()
                    .map_err(|_| CoreBridgeError::InvalidInstructionArgument)?,
                data,
            )
        }
        ProcessEncodedVaaDirective::VerifySignaturesV1 => match &ctx.accounts.guardian_set {
            Some(guardian_set) => {
                msg!("Directive: VerifySignaturesV1");
                verify_signatures_v1(vaa_acc_info, guardian_set)
            }
            _ => err!(ErrorCode::AccountNotEnoughKeys),
        },
    }
}

fn write_vaa(vaa_acc_info: &AccountInfo, index: usize, new_data: Vec<u8>) -> Result<()> {
    require!(
        !new_data.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let vaa_len = {
        let mut data: &[u8] = &vaa_acc_info.try_borrow_data()?;
        let header = ProcessingHeader::try_account_deserialize_unchecked(&mut data)?;
        require!(
            header.status == ProcessingStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        let vaa_len = u32::deserialize(&mut data)?;
        usize::try_from(vaa_len).unwrap()
    };

    let end = index.saturating_add(new_data.len());
    require_gte!(vaa_len, end, CoreBridgeError::DataOverflow);

    let data: &mut [u8] = &mut vaa_acc_info.try_borrow_mut_data()?;
    data[(START + index)..(START + end)].copy_from_slice(&new_data);

    // Done.
    Ok(())
}

fn verify_signatures_v1(
    vaa_acc_info: &AccountInfo<'_>,
    guardian_set: &Account<'_, GuardianSet>,
) -> Result<()> {
    let write_authority = {
        let mut data: &[u8] = &vaa_acc_info.try_borrow_data()?;
        data = &data[8..];

        let header = ProcessingHeader::deserialize(&mut data)?;
        require!(
            header.status == ProcessingStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        // Skip vaa length.
        data = &data[4..];

        // Parse and verify.
        let vaa = Vaa::parse(&mut data).map_err(|_| CoreBridgeError::CannotParseVaa)?;

        // Must be V1.
        require_eq!(
            vaa.version(),
            u8::from(VaaVersion::V1),
            CoreBridgeError::InvalidVaaVersion
        );

        // Make sure the encoded guardian set index agrees with the guardian set account's index.
        require_eq!(vaa.guardian_set_index(), guardian_set.index);

        // Do we have enough signatures for quorum?
        let guardian_keys = &guardian_set.keys;
        let quorum = crate::utils::quorum(guardian_keys.len());
        require_gte!(
            usize::from(vaa.signature_count()),
            quorum,
            CoreBridgeError::NoQuorum
        );

        // let sigs: Vec<_> = (0..num_signatures)
        //     .filter_map(|_| GuardianSetSig::read(&mut data).ok())
        //     .collect();
        // require!(
        //     usize::from(num_signatures) == sigs.len(),
        //     ErrorCode::AccountDidNotDeserialize
        // );

        // Generate the same message hash (using keccak) that the Guardians used to generate their
        // signatures. This message hash will be hashed again to produce the digest for
        // `secp256k1_recover`.
        let digest = vaa.body().double_digest();

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

        header.write_authority
    };

    let data: &mut [u8] = &mut vaa_acc_info.try_borrow_mut_data()?;
    let mut writer = std::io::Cursor::new(data);
    ProcessingHeader {
        status: ProcessingStatus::Verified,
        write_authority,
        version: VaaVersion::V1,
    }
    .try_account_serialize(&mut writer)
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
