use crate::{
    accounts::{
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadTransfer,
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
use solana_program::{
    account_info::AccountInfo,
    program::{
        invoke,
        invoke_signed,
    },
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};

#[derive(FromAccounts)]
pub struct CompleteNative<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub vaa: PayloadMessage<'b, PayloadTransfer>,
    pub claim: Mut<Claim<'b>>,
    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::MaybeInitialized }>>,
    pub to_authority: MaybeMut<Info<'b>>,
    pub custody: Mut<CustodyAccount<'b, { AccountState::Initialized }>>,
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,

    pub custody_signer: CustodySigner<'b>,
}

impl<'a> From<&CompleteNative<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteNative<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteNative<'a>> for CustodyAccountDerivationData {
    fn from(accs: &CompleteNative<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteNativeData {}

pub fn complete_native(
    ctx: &ExecutionContext,
    accs: &mut CompleteNative,
    _data: CompleteNativeData,
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
    if *accs.mint.info().key != accs.custody.mint {
        return Err(InvalidMint.into());
    }
    if *accs.custody_signer.key != accs.custody.owner {
        return Err(WrongAccountOwner.into());
    }

    // Verify VAA
    // Please refer to transfer.rs for why the token id is used to store the mint
    if accs.vaa.token_address != [1u8; 32] {
        return Err(InvalidMint.into());
    }
    let mut token_id_bytes = [0u8; 32];
    accs.vaa.token_id.to_big_endian(&mut token_id_bytes);
    if token_id_bytes != accs.mint.info().key.to_bytes() {
        return Err(InvalidMint.into());
    }
    if accs.vaa.token_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to != accs.to.info().key.to_bytes() {
        return Err(InvalidRecipient.into());
    }

    // Prevent vaa double signing
    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    if !accs.to.is_initialized() {
        let associated_addr = spl_associated_token_account::get_associated_token_address(
            accs.to_authority.info().key,
            accs.mint.info().key,
        );
        if *accs.to.info().key != associated_addr {
            return Err(InvalidAssociatedAccount.into());
        }
        // Create associated token account
        let ix = spl_associated_token_account::create_associated_token_account(
            accs.payer.info().key,
            accs.to_authority.info().key,
            accs.mint.info().key,
        );
        invoke(&ix, ctx.accounts)?;
    } else if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.custody.info().key,
        accs.to.info().key,
        accs.custody_signer.key,
        &[],
        1,
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    Ok(())
}

#[derive(FromAccounts)]
pub struct CompleteWrapped<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    // Signed message for the transfer
    pub vaa: PayloadMessage<'b, PayloadTransfer>,
    pub claim: Mut<Claim<'b>>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Mut<Data<'b, SplAccount, { AccountState::MaybeInitialized }>>,
    pub to_authority: MaybeMut<Info<'b>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::MaybeInitialized }>>,
    pub meta: Mut<WrappedTokenMeta<'b, { AccountState::MaybeInitialized }>>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CompleteWrapped<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteWrapped<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &CompleteWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
            token_id: accs.vaa.token_id,
        }
    }
}

impl<'a> From<&CompleteWrapped<'a>> for WrappedMetaDerivationData {
    fn from(accs: &CompleteWrapped<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteWrappedData {}

pub fn complete_wrapped(
    ctx: &ExecutionContext,
    accs: &mut CompleteWrapped,
    _data: CompleteWrappedData,
) -> Result<()> {
    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mint
    let derivation_data: WrappedDerivationData = (&*accs).into();
    accs.mint
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify VAA
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }
    if accs.vaa.to != accs.to.info().key.to_bytes() {
        return Err(InvalidRecipient.into());
    }

    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    // Initialize the NFT if it doesn't already exist
    if !accs.meta.is_initialized() {
        // Create mint account
        accs.mint
            .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

        // Initialize mint
        let init_ix = spl_token::instruction::initialize_mint(
            &spl_token::id(),
            accs.mint.info().key,
            accs.mint_authority.key,
            None,
            0,
        )?;
        invoke_signed(&init_ix, ctx.accounts, &[])?;

        // Create meta account
        accs.meta
            .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

        // Populate meta account
        accs.meta.chain = accs.vaa.token_chain;
        accs.meta.token_address = accs.vaa.token_address;
        accs.meta.token_id = accs.vaa.token_id.0;
    }

    if !accs.to.is_initialized() {
        let associated_addr = spl_associated_token_account::get_associated_token_address(
            accs.to_authority.info().key,
            accs.mint.info().key,
        );
        if *accs.to.info().key != associated_addr {
            return Err(InvalidAssociatedAccount.into());
        }
        // Create associated token account
        let ix = spl_associated_token_account::create_associated_token_account(
            accs.payer.info().key,
            accs.to_authority.info().key,
            accs.mint.info().key,
        );
        invoke_signed(&ix, ctx.accounts, &[])?;
    } else if *accs.mint.info().key != accs.to.mint {
        return Err(InvalidMint.into());
    }

    // Mint tokens
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        accs.mint.info().key,
        accs.to.info().key,
        accs.mint_authority.key,
        &[],
        1,
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.mint_authority, None)?;

    Ok(())
}

#[derive(FromAccounts)]
pub struct CompleteWrappedMeta<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    // VAA for the transfer; this does not need to get claimed
    pub vaa: PayloadMessage<'b, PayloadTransfer>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub mint: WrappedMint<'b, { AccountState::Initialized }>,
    pub meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,

    /// SPL Metadata for the associated Mint
    pub spl_metadata: Mut<SplTokenMeta<'b>>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CompleteWrappedMeta<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteWrappedMeta<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteWrappedMeta<'a>> for WrappedDerivationData {
    fn from(accs: &CompleteWrappedMeta<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
            token_id: accs.vaa.token_id,
        }
    }
}

impl<'a> From<&CompleteWrappedMeta<'a>> for WrappedMetaDerivationData {
    fn from(accs: &CompleteWrappedMeta<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteWrappedMetaData {}

pub fn complete_wrapped_meta(
    ctx: &ExecutionContext,
    accs: &mut CompleteWrappedMeta,
    _data: CompleteWrappedMetaData,
) -> Result<()> {
    use bstr::ByteSlice;

    // Verify the chain registration
    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify mint
    let derivation_data: WrappedDerivationData = (&*accs).into();
    accs.mint
        .verify_derivation(ctx.program_id, &derivation_data)?;

    // Verify VAA
    if accs.vaa.to_chain != CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    // Make sure the metadata hasn't been initialized yet
    if !accs.spl_metadata.data_is_empty() {
        return Err(AlreadyExecuted.into());
    }

    // Initialize spl meta
    accs.spl_metadata.verify_derivation(
        &spl_token_metadata::id(),
        &SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        },
    )?;

    let name = accs.vaa.name.clone();
    let mut symbol: Vec<u8> = accs.vaa.symbol.clone().as_bytes().to_vec();
    symbol.truncate(10);
    let mut symbol: Vec<char> = symbol.chars().collect();
    symbol.retain(|&c| c != '\u{FFFD}');
    let symbol: String = symbol.iter().collect();

    let spl_token_metadata_ix = spl_token_metadata::instruction::create_metadata_accounts(
        spl_token_metadata::id(),
        *accs.spl_metadata.key,
        *accs.mint.info().key,
        *accs.mint_authority.info().key,
        *accs.payer.info().key,
        *accs.mint_authority.info().key,
        name,
        symbol,
        accs.vaa.uri.clone(),
        None,
        0,
        false,
        true,
    );
    invoke_seeded(&spl_token_metadata_ix, ctx, &accs.mint_authority, None)?;

    Ok(())
}
