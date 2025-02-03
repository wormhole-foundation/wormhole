use anchor_lang::{
    prelude::*,
    solana_program::{
        keccak::{self, HASH_BYTES},
        program_memory::sol_memcpy,
        secp256k1_recover::secp256k1_recover,
    },
};
use wormhole_raw_vaas::{utils::quorum, GuardianSetSig};
use wormhole_svm_definitions::CORE_BRIDGE_PROGRAM_ID;

use crate::{
    error::WormholeVerifyVaaShim,
    state::{GuardianSignatures, WormholeGuardianSet},
};

#[derive(Accounts)]
#[instruction(guardian_set_bump: u8)]
pub struct VerifyHash<'info> {
    /// Guardian set used for signature verification.
    #[account(
        seeds = [
            WormholeGuardianSet::SEED_PREFIX,
            guardian_signatures.guardian_set_index_be.as_ref()
        ],
        bump = guardian_set_bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    guardian_set: Account<'info, WormholeGuardianSet>,

    /// Stores unverified guardian signatures as they are too large to fit in the instruction data.
    guardian_signatures: Account<'info, GuardianSignatures>,
}

impl<'info> VerifyHash<'info> {
    pub fn constraints(ctx: &Context<Self>, digest: &[u8; HASH_BYTES]) -> Result<()> {
        let guardian_set = &ctx.accounts.guardian_set;

        // Check that the guardian set is still active.
        let timestamp = Clock::get()?
            .unix_timestamp
            .try_into()
            .expect("timestamp overflow");
        require!(
            guardian_set.is_active(&timestamp),
            WormholeVerifyVaaShim::GuardianSetExpired
        );

        let guardian_signatures = &ctx.accounts.guardian_signatures.guardian_signatures;

        // This section is borrowed from https://github.com/wormhole-foundation/wormhole/blob/wen/solana-rewrite/solana/programs/core-bridge/src/processor/parse_and_verify_vaa/verify_encoded_vaa_v1.rs#L72-L103
        // Also similarly used here https://github.com/pyth-network/pyth-crosschain/blob/6771c2c6998f53effee9247347cb0ac71612b3dc/target_chains/solana/programs/pyth-solana-receiver/src/lib.rs#L121-L159
        // Do we have enough signatures for quorum?
        let guardian_keys = &guardian_set.keys;
        let quorum = quorum(guardian_keys.len());
        require!(
            guardian_signatures.len() >= quorum,
            WormholeVerifyVaaShim::NoQuorum
        );

        // Verify signatures
        let mut last_guardian_index = None;
        for sig_bytes in guardian_signatures {
            let sig = GuardianSetSig::try_from(sig_bytes.as_slice())
                .map_err(|_| WormholeVerifyVaaShim::InvalidSignature)?;
            // We do not allow for non-increasing guardian signature indices.
            let index = usize::from(sig.guardian_index());
            if let Some(last_index) = last_guardian_index {
                require!(
                    index > last_index,
                    WormholeVerifyVaaShim::InvalidGuardianIndexNonIncreasing
                );
            }

            // Does this guardian index exist in this guardian set?
            let guardian_pubkey = guardian_keys
                .get(index)
                .ok_or_else(|| error!(WormholeVerifyVaaShim::InvalidGuardianIndexOutOfRange))?;

            // Now verify that the signature agrees with the expected Guardian's pubkey.
            verify_guardian_signature(&sig, guardian_pubkey, digest)?;

            last_guardian_index = Some(index);
        }
        // End borrowed section

        // Done.
        Ok(())
    }
}

#[access_control(VerifyHash::constraints(&ctx, &digest))]
pub fn verify_hash(
    ctx: Context<VerifyHash>,
    _guardian_set_bump: u8,
    digest: [u8; HASH_BYTES],
) -> Result<()> {
    Ok(())
}

/**
 * Borrowed from https://github.com/wormhole-foundation/wormhole/blob/wen/solana-rewrite/solana/programs/core-bridge/src/processor/parse_and_verify_vaa/verify_encoded_vaa_v1.rs#L121
 * Also used here https://github.com/pyth-network/pyth-crosschain/blob/6771c2c6998f53effee9247347cb0ac71612b3dc/target_chains/solana/programs/pyth-solana-receiver/src/lib.rs#L432
 */
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
            .map_err(|_| WormholeVerifyVaaShim::InvalidSignature)?;

        // The Ethereum public key is the last 20 bytes of keccak hashed public key above.
        let hashed = keccak::hash(&pubkey.to_bytes());

        let mut eth_pubkey = [0; 20];
        sol_memcpy(&mut eth_pubkey, &hashed.0[12..], 20);

        eth_pubkey
    };

    // The recovered public key should agree with the Guardian's public key at this index.
    require!(
        recovered == *guardian_pubkey,
        WormholeVerifyVaaShim::InvalidGuardianKeyRecovery
    );

    // Done.
    Ok(())
}
