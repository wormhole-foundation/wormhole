use crate::{
    error::CoreBridgeError,
    legacy::{
        instruction::PostVaaArgs,
        utils::{AccountVariant, LegacyAnchorized},
    },
    state::{GuardianSet, PostedVaaV1, PostedVaaV1Info, SignatureSet},
    types::MessageHash,
    utils,
};
use anchor_lang::prelude::*;

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
#[instruction(args: PostVaaArgs)]
pub struct PostVaa<'info> {
    /// Guardian set used for signature verification (whose index should agree with the signature
    /// set account's guardian set index).
    #[account(
        seeds = [
            GuardianSet::SEED_PREFIX,
            signature_set.guardian_set_index.to_be_bytes().as_ref()
        ],
        bump,
    )]
    guardian_set: Account<'info, AccountVariant<GuardianSet>>,

    /// CHECK: Core Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// Signature set, which stores signature validation from Sig Verify native program.
    signature_set: Account<'info, SignatureSet>,

    /// Posted VAA created by this instruction handler.
    ///
    /// NOTE: This instruction handler previously handled the case where this account was created
    /// already, where the handler would bail out with success.
    #[account(
        init,
        payer = payer,
        space = PostedVaaV1::compute_size(args.payload.len()),
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            signature_set.message_hash.as_ref()
        ],
        bump,
    )]
    posted_vaa: Account<'info, LegacyAnchorized<PostedVaaV1>>,

    #[account(mut)]
    payer: Signer<'info>,

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _clock: UncheckedAccount<'info>,

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostVaaArgs> for PostVaa<'info> {
    const LOG_IX_NAME: &'static str = "LegacyPostVaa";

    const ANCHOR_IX_FN: fn(Context<Self>, PostVaaArgs) -> Result<()> = post_vaa;
}

impl<'info> PostVaa<'info> {
    pub fn constraints(ctx: &Context<Self>, args: &PostVaaArgs) -> Result<()> {
        let guardian_set = ctx.accounts.guardian_set.inner();

        // Check that the guardian set is still active.
        let timestamp = Clock::get().map(Into::into)?;
        require!(
            guardian_set.is_active(&timestamp),
            CoreBridgeError::GuardianSetExpired
        );

        let signature_set = &ctx.accounts.signature_set;
        require!(
            !INVALID_SIGNATURE_SET_KEYS.contains(&signature_set.key().to_string().as_str()),
            CoreBridgeError::InvalidSignatureSet
        );

        // Number of verified signatures in the signature set account must be at least quorum with
        // the guardian set.
        require!(
            signature_set.num_verified() >= utils::quorum(guardian_set.keys.len()),
            CoreBridgeError::NoQuorum
        );

        // Recompute the message hash and compare it to the one in the signature set account.
        let recomputed = solana_program::keccak::hashv(&[
            &args.timestamp.to_be_bytes(),
            &args.nonce.to_be_bytes(),
            &args.emitter_chain.to_be_bytes(),
            &args.emitter_address,
            &args.sequence.to_be_bytes(),
            &[args.consistency_level],
            &args.payload,
        ]);
        require_eq!(
            MessageHash::from(recomputed),
            signature_set.message_hash,
            CoreBridgeError::InvalidMessageHash
        );

        // Done.
        Ok(())
    }
}

/// Processor to write a verified VAA to a `PostedVaaV1` account. This instruction handler requires
/// that the number of verified signers in the [SignatureSet] account is at least the quorum using
/// the guardian set, whose index is encoded in this account. And the message hash in this account
/// must agree with the recomputed one using this instruction handler's arguments.
///
/// NOTE: It is recommended that VAAs be verified using the new Anchor instructions
/// `init_encoded_vaa` and `process_encoded_vaa`, which does not rely on the Sig Verify native
/// program to verify elliptic curve signatures.
#[access_control(PostVaa::constraints(&ctx, &args))]
fn post_vaa(ctx: Context<PostVaa>, args: PostVaaArgs) -> Result<()> {
    let PostVaaArgs {
        _gap_0,
        timestamp,
        nonce,
        emitter_chain,
        emitter_address,
        sequence,
        consistency_level,
        payload,
    } = args;

    // Set the posted VAA account with this instruction data.
    let signature_set = &ctx.accounts.signature_set;
    ctx.accounts.posted_vaa.set_inner(
        PostedVaaV1 {
            info: PostedVaaV1Info {
                consistency_level,
                timestamp: timestamp.into(),
                signature_set: signature_set.key(),
                guardian_set_index: signature_set.guardian_set_index,
                nonce,
                sequence,
                emitter_chain,
                emitter_address,
            },
            payload,
        }
        .into(),
    );

    // Done.
    Ok(())
}
