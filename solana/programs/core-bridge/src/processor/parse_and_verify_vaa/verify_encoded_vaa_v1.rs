use crate::{
    error::CoreBridgeError,
    legacy::utils::AccountVariant,
    state::{GuardianSet, Header, ProcessingStatus},
    zero_copy::EncodedVaa,
};
use anchor_lang::prelude::*;
use solana_program::{keccak, program_memory::sol_memcpy, secp256k1_recover::secp256k1_recover};
use wormhole_raw_vaas::{GuardianSetSig, Vaa};

#[derive(Accounts)]
pub struct VerifyEncodedVaaV1<'info> {
    write_authority: Signer<'info>,

    /// CHECK: The encoded VAA account, which stores the VAA buffer. This buffer must first be
    /// written to and then verified.
    #[account(mut)]
    encoded_vaa: AccountInfo<'info>,

    /// Guardian set account, which should be the same one that was used to attest for the VAA. The
    /// signatures in the encoded VAA are verified against this guardian set.
    #[account(
        seeds = [
            GuardianSet::SEED_PREFIX,
            guardian_set.inner().index.to_be_bytes().as_ref()
        ],
        bump,
    )]
    guardian_set: Account<'info, AccountVariant<GuardianSet>>,
}

impl<'info> VerifyEncodedVaaV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Guardian set must be active.
        let timestamp = Clock::get().map(Into::into)?;
        require!(
            ctx.accounts.guardian_set.inner().is_active(&timestamp),
            CoreBridgeError::GuardianSetExpired
        );

        // Check write authority.
        let vaa = EncodedVaa::parse_unverified(&ctx.accounts.encoded_vaa)?;
        require_keys_eq!(
            ctx.accounts.write_authority.key(),
            vaa.write_authority(),
            CoreBridgeError::WriteAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

#[access_control(VerifyEncodedVaaV1::constraints(&ctx))]
pub fn verify_encoded_vaa_v1(ctx: Context<VerifyEncodedVaaV1>) -> Result<()> {
    let guardian_set = &ctx.accounts.guardian_set.inner();

    let write_authority = {
        let encoded_vaa = EncodedVaa::parse_unverified(&ctx.accounts.encoded_vaa).unwrap();
        require!(
            encoded_vaa.status() == ProcessingStatus::Writing,
            CoreBridgeError::VaaAlreadyVerified
        );

        // Parse and verify.
        let vaa =
            Vaa::parse(encoded_vaa.buf()).map_err(|_| error!(CoreBridgeError::CannotParseVaa))?;

        // Must be V1.
        require_eq!(vaa.version(), 1, CoreBridgeError::InvalidVaaVersion);

        // Make sure the encoded guardian set index agrees with the guardian set account's index.
        require_eq!(
            vaa.guardian_set_index(),
            guardian_set.index,
            CoreBridgeError::GuardianSetMismatch
        );

        // Do we have enough signatures for quorum?
        let guardian_keys = &guardian_set.keys;
        let quorum = crate::utils::quorum(guardian_keys.len());
        require!(
            usize::from(vaa.signature_count()) >= quorum,
            CoreBridgeError::NoQuorum
        );

        // Generate the same message hash (using keccak) that the Guardians used to generate their
        // signatures. This message hash will be hashed again to produce the digest for
        // `secp256k1_recover`.
        let digest = keccak::hash(keccak::hash(vaa.body().as_ref()).as_ref());

        // Only verify as many as we need (up to quorum).
        let mut last_guardian_index = None;
        for sig in vaa.signatures() {
            // We do not allow for non-increasing guardian signature indices.
            let index = usize::from(sig.guardian_index());
            if let Some(last_index) = last_guardian_index {
                require!(index > last_index, CoreBridgeError::InvalidGuardianIndex);
            }

            // Does this guardian index exist in this guardian set?
            let guardian_pubkey = guardian_keys
                .get(index)
                .ok_or_else(|| error!(CoreBridgeError::InvalidGuardianIndex))?;

            // Now verify that the signature agrees with the expected Guardian's pubkey.
            verify_guardian_signature(&sig, guardian_pubkey, digest.as_ref())?;

            last_guardian_index = Some(index);
        }

        encoded_vaa.write_authority()
    };

    let acc_data: &mut [_] = &mut ctx.accounts.encoded_vaa.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);
    (
        EncodedVaa::DISC,
        Header {
            status: ProcessingStatus::Verified,
            write_authority,
            version: 1,
        },
    )
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
        sol_memcpy(&mut eth_pubkey, &hashed.0[12..], 20);

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
