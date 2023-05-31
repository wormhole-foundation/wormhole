use crate::{
    error::CoreBridgeError,
    legacy::instruction::LegacyPostVaaArgs,
    state::{GuardianSet, PostedVaaV1Bytes, PostedVaaV1Metadata, SignatureSet},
    types::MessageHash,
    utils,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{NewAccountSize, SeedPrefix};

/// Invalidated signature sets.
///
/// NOTE: When `post_vaa` is deprecated, we can remove these because `SignatureSet` will not be used
/// any longer.
const INVALID_SIGNATURE_SET_KEYS: [&str; 16] = [
    "18eK1799CaNMGCUnnCt1Kq2uwKkax6T2WmtrDsZuVFQ",
    "2g6NCUUPaD6AxdHPQMVLpjpAvBfKMek6dDiGUe2A6T33",
    "3hYV5968hNzbqUfcvnQ6v9D5h32hEwGJn19c47N3unNj",
    "76eEyhaEKs4mesjiQiu8bghvwDHNxJW3EfcpbNC78y1z",
    "7PdcxSn7xk2UN5VYmKnJ2Q64PdBhbBQFf4RwHqhQCMgv",
    "94wXN3z3Pph2vMVaviZSouo7oCDqt4fekvqT3FYJSrWA",
    "AXe9VXd9jjXkBxSdvgj4bHSZNeqxY73sSQEsp1tnekY4",
    "B2hS49B8n4Ad6cxZLoAjz7Hux7Kf17D5xUX3neDPHpug",
    "BTXnYYjnfXByqJprarqzp65Yha2XwQVmg8V8KWBhr6aA",
    "Bzb5G4Y8QcaMVMQq3r8q1SuKSxtgnWSFdKCEisJCbcBP",
    "CJfRUQxyonG6B5mnztsNUqxknbFT89DJdrdrzV9F96mU",
    "CK1j9TxWP1T5w1QzFu4vPDAbUR34mfVqvk5wziE8TzST",
    "E8qKJMwzBCiHCHUmBEcL631kN5CjfsHNx24osFLfHg69",
    "EtMw1nQ4AQaH53RjYz3pRk12rrqWjcYjPDETphYJzmCX",
    "EVNwqfgkUnJoMqBqiHgDfa3TLZPQocX1hpcbAXbpcSLv",
    "FixSiDfTxvoy5Zgjp5KdFU8U23ChwCxPWY3WTkmMW2fU",
];

#[derive(Accounts)]
#[instruction(args: LegacyPostVaaArgs)]
pub struct PostVaa<'info> {
    /// Guardian set used for signature verification.
    #[account(
        seeds = [GuardianSet::seed_prefix(), &signature_set.guardian_set_index.to_be_bytes()],
        bump,
    )]
    guardian_set: Account<'info, GuardianSet>,

    /// CHECK: Core Bridge never needed this account for this instruction.
    _bridge: UncheckedAccount<'info>,

    /// Signature set, which stores signature validation from libsecp256k1 program.
    ///
    /// NOTE: We prefer to make this account mutable so we have the ability to close this account
    /// once this VAA is posted. But we are prserving read-only to not alter the existing behavior.
    signature_set: Account<'info, SignatureSet>,

    /// Posted verified message. This account is created if it hasn't been created already.
    ///
    /// NOTE: This instruction handler previously handled the case where this account was created
    /// already, where the handler would bail out with success.
    #[account(
        init,
        payer = payer,
        space = PostedVaaV1Bytes::compute_size(args.payload.len()),
        seeds = [PostedVaaV1Bytes::seed_prefix(), signature_set.message_hash.as_ref()],
        bump,
    )]
    posted_vaa: Account<'info, PostedVaaV1Bytes>,

    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> PostVaa<'info> {
    pub fn accounts(ctx: &Context<Self>, args: &LegacyPostVaaArgs) -> Result<()> {
        let signature_set = &ctx.accounts.signature_set;
        require!(
            !INVALID_SIGNATURE_SET_KEYS.contains(&signature_set.key().to_string().as_str()),
            CoreBridgeError::InvalidSignatureSet
        );

        require_gte!(
            signature_set.num_verified(),
            utils::quorum(ctx.accounts.guardian_set.keys.len()),
            CoreBridgeError::NoQuorum
        );

        let recomputed = utils::compute_message_hash(
            args.timestamp.into(),
            args.nonce,
            args.emitter_chain,
            &args.emitter_address,
            args.sequence,
            args.consistency_level,
            &args.payload,
        );
        require_eq!(
            MessageHash::from(recomputed),
            signature_set.message_hash,
            CoreBridgeError::InvalidMessageHash
        );

        // Done.
        Ok(())
    }
}

#[access_control(PostVaa::accounts(&ctx, &args))]
pub fn post_vaa(ctx: Context<PostVaa>, args: LegacyPostVaaArgs) -> Result<()> {
    let LegacyPostVaaArgs {
        _version,
        _guardian_set_index,
        timestamp,
        nonce,
        emitter_chain,
        emitter_address,
        sequence,
        consistency_level,
        payload,
    } = args;

    // Set the `message` account with this instruction data.
    ctx.accounts.posted_vaa.set_inner(PostedVaaV1Bytes {
        meta: PostedVaaV1Metadata {
            consistency_level,
            timestamp: timestamp.into(),
            signature_set: ctx.accounts.signature_set.key(),
            guardian_set_index: ctx.accounts.guardian_set.index,
            nonce,
            sequence,
            emitter_chain,
            emitter_address,
        },
        payload,
    });

    // Done.
    Ok(())
}
