use crate::{
    accounts::{
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadTransferWithPayload,
    types::*,
    TokenBridgeError::*,
};
use bridge::{
    accounts::claim::{
        self,
        Claim,
    },
    PayloadMessage,
    CHAIN_ID_SOLANA,
};
use solana_program::account_info::AccountInfo;
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    *,
};

use solana_program::pubkey::Pubkey;

////////////////////////////////////////////////////////////////////////////////
// Recipient

#[repr(transparent)]
pub struct RedeemerAccount<'b>(pub MaybeMut<Signer<Info<'b>>>);

impl<'a, 'b: 'a> Peel<'a, 'b> for RedeemerAccount<'b> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self>
    where
        Self: Sized,
    {
        Ok(RedeemerAccount(MaybeMut::peel(ctx)?))
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        MaybeMut::persist(&self.0, program_id)
    }
}

// May or may not be a PDA, so we don't use [`Derive`], instead implement
// [`Seeded`] directly.
impl<'b> Seeded<()> for RedeemerAccount<'b> {
    fn seeds(_accs: ()) -> Vec<Vec<u8>> {
        vec![String::from("redeemer").as_bytes().to_vec()]
    }
}

impl<'a, 'b: 'a> Keyed<'a, 'b> for RedeemerAccount<'b> {
    fn info(&'a self) -> &Info<'b> {
        &self.0
    }
}

impl<'b> RedeemerAccount<'b> {
    /// Transfer with payload can only be redeemed by the recipient. The idea is
    /// to target contracts which can then decide how to process the payload.
    ///
    /// The actual recipient (the `to` field in the VAA) may be either a wallet
    /// or a program id. Since wallets can sign transactions directly, if the
    /// recipient is a wallet, then we just require the wallet to sign the
    /// redeem transaction. If, however, the recipient is a program, then
    /// program can only provide a PDA as a signer. In this case, we require the
    /// this to be a PDA derived from the recipient program id and the string
    /// "redeemer".
    ///
    /// That is, the redeemer account either matches the `vaa.to` field directly
    /// (user wallets), or is a PDA derived from vaa.to and "sender" (contracts).
    ///
    /// The `vaa.to` account must own the token account.
    fn verify_recipient_address(&self, recipient: &Pubkey) -> Result<()> {
        if recipient == self.info().key {
            return Ok(());
        } else {
            self.verify_derivation(recipient, ())
        }
    }
}

#[derive(FromAccounts)]
pub struct CompleteNativeWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub vaa: PayloadMessage<'b, PayloadTransferWithPayload>,
    pub claim: Mut<Claim<'b>>,
    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,

    /// See [`verify_recipient_address`]
    pub redeemer: RedeemerAccount<'b>,
    pub to_fees: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub custody: Mut<CustodyAccount<'b, { AccountState::Initialized }>>,
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,

    pub custody_signer: CustodySigner<'b>,
}

impl<'a> From<&CompleteNativeWithPayload<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteNativeWithPayload<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteNativeWithPayload<'a>> for CustodyAccountDerivationData {
    fn from(accs: &CompleteNativeWithPayload<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteNativeWithPayloadData {}

pub fn complete_native_with_payload(
    ctx: &ExecutionContext,
    accs: &mut CompleteNativeWithPayload,
    _data: CompleteNativeWithPayloadData,
) -> Result<()> {
    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify that the custody account is derived correctly
    let derivation_data: CustodyAccountDerivationData = (&*accs).into();
    accs.custody
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mints
    if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.to_fees.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.custody.mint {
        return Err(InvalidMint.into());
    }
    if *accs.custody_signer.key != accs.custody.owner {
        return Err(WrongAccountOwner.into());
    }

    // Verify VAA
    if accs.vaa.token_address != accs.mint.info().key.to_bytes() {
        return Err(InvalidMint.into());
    }
    if accs.vaa.token_chain != 1 {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let recipient = Pubkey::try_from_slice(&accs.vaa.to)?;
    accs.redeemer.verify_recipient_address(&recipient)?;

    // Token account owner must be either the VAA-specified recipient, or the
    // redeemer account (for regular wallets, these two are equal, for programs
    // the latter is a PDA)
    if recipient != accs.to.owner && *accs.redeemer.info().key != accs.to.owner {
        return Err(InvalidRecipient.into());
    }

    // Prevent vaa double signing
    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    let mut amount = accs.vaa.amount.as_u64();

    // Wormhole always caps transfers at 8 decimals; un-truncate if the local token has more
    if accs.mint.decimals > 8 {
        amount *= 10u64.pow((accs.mint.decimals - 8) as u32);
    }

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.custody.info().key,
        accs.to.info().key,
        accs.custody_signer.key,
        &[],
        amount,
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    Ok(())
}

#[derive(FromAccounts)]
pub struct CompleteWrappedWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    /// Signed message for the transfer
    pub vaa: PayloadMessage<'b, PayloadTransferWithPayload>,
    pub claim: Mut<Claim<'b>>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,

    /// See [`verify_recipient_address`]
    pub redeemer: RedeemerAccount<'b>,
    pub to_fees: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::Initialized }>>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CompleteWrappedWithPayload<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteWrappedWithPayload<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteWrappedWithPayload<'a>> for WrappedDerivationData {
    fn from(accs: &CompleteWrappedWithPayload<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteWrappedWithPayloadData {}

pub fn complete_wrapped_with_payload(
    ctx: &ExecutionContext,
    accs: &mut CompleteWrappedWithPayload,
    _data: CompleteWrappedWithPayloadData,
) -> Result<()> {
    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mint
    accs.wrapped_meta.verify_derivation(
        ctx.program_id,
        &WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        },
    )?;
    if accs.wrapped_meta.token_address != accs.vaa.token_address
        || accs.wrapped_meta.chain != accs.vaa.token_chain
    {
        return Err(InvalidMint.into());
    }

    // Verify mints
    if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }
    if *accs.mint.info().key != accs.to_fees.mint {
        return Err(InvalidMint.into());
    }

    // Verify VAA
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let recipient = Pubkey::try_from_slice(&accs.vaa.to)?;
    accs.redeemer.verify_recipient_address(&recipient)?;

    // Token account owner must be either the VAA-specified recipient, or the
    // redeemer account (for regular wallets, these two are equal, for programs
    // the latter is a PDA)
    if recipient != accs.to.owner && *accs.redeemer.info().key != accs.to.owner {
        return Err(InvalidRecipient.into());
    }

    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    // Mint tokens
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        accs.mint.info().key,
        accs.to.info().key,
        accs.mint_authority.key,
        &[],
        accs.vaa.amount.as_u64(),
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.mint_authority, None)?;

    Ok(())
}
