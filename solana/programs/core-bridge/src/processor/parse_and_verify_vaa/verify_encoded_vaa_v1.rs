use crate::{
    error::CoreBridgeError,
    legacy::utils::AccountVariant,
    state::{EncodedVaa, GuardianSet, ProcessingStatus},
};
use anchor_lang::prelude::*;
use solana_program::{keccak, program_memory::sol_memcpy, secp256k1_recover::secp256k1_recover};
use wormhole_raw_vaas::{GuardianSetSig, Vaa};

#[derive(Accounts)]
pub struct VerifyEncodedVaaV1<'info> {
    write_authority: Signer<'info>,

    /// CHECK: The encoded VAA account, which stores the VAA buffer. This buffer must first be
    /// written to and then verified.
    #[account(
        mut,
        owner = crate::ID,
        constraint = EncodedVaa::require_draft_vaa(&draft_vaa, &write_authority)?
    )]
    draft_vaa: AccountInfo<'info>,

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

        // Done.
        Ok(())
    }
}

#[access_control(VerifyEncodedVaaV1::constraints(&ctx))]
pub fn verify_encoded_vaa_v1(ctx: Context<VerifyEncodedVaaV1>) -> Result<()> {
    let mut header = EncodedVaa::try_deserialize_header(&ctx.accounts.draft_vaa)?;

    // Verify signatures in encoded VAA against the guardian pubkeys in the guardian set.
    {
        let mut acc_data: &[_] = &ctx.accounts.draft_vaa.data.borrow();
        acc_data = &acc_data[EncodedVaa::VAA_START..];

        // Parse and verify.
        let vaa = Vaa::parse(acc_data).map_err(|_| error!(CoreBridgeError::CannotParseVaa))?;

        // Must be V1.
        require_eq!(vaa.version(), 1, CoreBridgeError::InvalidVaaVersion);

        // Make sure the encoded guardian set index agrees with the guardian set account's index.
        let guardian_set = ctx.accounts.guardian_set.inner();
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
    }

    // Revise the header.
    header.status = ProcessingStatus::Verified;
    header.version = 1;

    // Finally serialize.
    let acc_data: &mut [_] = &mut ctx.accounts.draft_vaa.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);
    (
        <EncodedVaa as anchor_lang::Discriminator>::DISCRIMINATOR,
        header,
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
