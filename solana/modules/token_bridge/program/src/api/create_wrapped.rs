use crate::{
    accounts::{
        ConfigAccount,
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
    messages::PayloadAssetMeta,
    types::*,
    TokenBridgeError::InvalidMint,
};
use bridge::{
    api::ForeignAddress,
    vaa::ClaimableVAA,
};
use phf::phf_map;
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};
use spl_token::{
    error::TokenError::OwnerMismatch,
    state::{
        Account,
        Mint,
    },
};
use std::{
    cmp::min,
    convert::TryInto,
    ops::{
        Deref,
        DerefMut,
    },
    str::FromStr,
};

#[derive(FromAccounts)]
pub struct CreateWrapped<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,
    pub vaa: ClaimableVAA<'b, PayloadAssetMeta>,

    // New Wrapped
    pub mint: Mut<WrappedMint<'b, { AccountState::MaybeInitialized }>>,
    pub meta: Mut<WrappedTokenMeta<'b, { AccountState::Uninitialized }>>,

    /// SPL Metadata for the associated Mint
    pub spl_metadata: Mut<SplTokenMeta<'b>>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CreateWrapped<'a>> for EndpointDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CreateWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
        }
    }
}

impl<'a> From<&CreateWrapped<'a>> for WrappedMetaDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

impl<'b> InstructionContext<'b> for CreateWrapped<'b> {
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CreateWrappedData {}

#[cfg(not(feature = "test"))]
pub static SOLLET_MINTS: phf::Map<&str, (u16, &str)> = phf_map! {
        // "WETH",
        "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" => (2, "2FPyTwcZLUg1MDrwsyoP4D6s1tM7hAkHYRjkNb5w6Pxk"),
        // "YFI",
        "0000000000000000000000000bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e" => (2, "3JSf5tPeuscJGtaCp5giEiDhv51gQ4v3zWg8DGgyLfAB"),
        // "LINK",
        "000000000000000000000000514910771af9ca656af840dff83e8264ecf986ca" => (2, "CWE8jPTUYhdCTZYWPTe1o5DFqfdjzWKc9WKz6rSjQUdG"),
        // "SUSHI",
        "0000000000000000000000006b3595068778dd592e39a122f4f5a5cf09c90fe2" => (2, "AR1Mtgh7zAtxuxGd2XPovXPVjcSdY3i4rQYisNadjfKy"),
        // "ALEPH",
        "00000000000000000000000027702a26126e0B3702af63Ee09aC4d1A084EF628" => (2, "CsZ5LZkDS7h9TDKjrbL7VAwQZ9nsRu8vJLhRYfmGaN8K"),
        // "SXP",
        "0000000000000000000000008ce9137d39326ad0cd6491fb5cc0cba0e089b6a9" => (2, "SF3oTvfWzEP3DTwGSvUXRrGTvr75pdZNnBLAH9bzMuX"),
        // "CREAM",
        "0000000000000000000000002ba592F78dB6436527729929AAf6c908497cB200" => (2, "5Fu5UUgbjpUvdBveb3a1JTNirL8rXtiYeSMWvKjtUNQv"),
        // "FRONT",
        "000000000000000000000000f8C3527CC04340b208C854E985240c02F7B7793f" => (2, "9S4t2NEAiJVMvPdRYKVrfJpBafPBLtvbvyS3DecojQHw"),
        // "AKRO",
        "0000000000000000000000008ab7404063ec4dbcfd4598215992dc3f8ec853d7" => (2, "6WNVCuxCGJzNjmMZoKyhZJwvJ5tYpsLyAtagzYASqBoF"),
        // "HXRO",
        "0000000000000000000000004bd70556ae3f8a6ec6c4080a0c327b24325438f3" => (2, "DJafV9qemGp7mLMEn5wrfqaFwxsbLgUsGVS16zKRk9kc"),
        // "UNI",
        "0000000000000000000000001f9840a85d5af5bf1d1762f925bdaddc4201f984" => (2, "DEhAasscXF4kEGxFgJ3bq4PpVGp5wyUxMRvn6TzGVHaw"),
        // "FTT",
        "00000000000000000000000050d1c9771902476076ecfc8b2a83ad6b9355a4c9" => (2, "AGFEad2et2ZJif9jaGpdMixQqvW5i81aBdvKe7PHNfz3"),
        // "LUA",
        "000000000000000000000000b1f66997a5760428d3a87d68b90bfe0ae64121cc" => (2, "EqWCKXfs3x47uVosDpTRgFniThL9Y8iCztJaapxbEaVX"),
        // "MATH",
        "00000000000000000000000008d967bb0134f2d07f7cfb6e246680c53927dd30" => (2, "GeDS162t9yGJuLEHPWXXGrb1zwkzinCgRwnT8vHYjKza"),
        // "KEEP",
        "00000000000000000000000085eee30c52b0b379b046fb0f85f4f3dc3009afec" => (2, "GUohe4DJUA5FKPWo3joiPgsB7yzer7LpDmt1Vhzy3Zht"),
        // "SWAG",
        "00000000000000000000000087eDfFDe3E14c7a66c9b9724747a1C5696b742e6" => (2, "9F9fNTT6qwjsu4X4yWYKZpsbw5qT7o6yR2i57JF2jagy"),
        // "CEL",
        "000000000000000000000000aaaebe6fe48e54f431b0c390cfaf0b017d09d42d" => (2, "DgHK9mfhMtUwwv54GChRrU54T2Em5cuszq2uMuen1ZVE"),
        // "RSR",
        "0000000000000000000000008762db106b2c2a0bccb3a80d1ed41273552616e8" => (2, "7ncCLJpP3MNww17LW8bRvx8odQQnubNtfNZBL5BgAEHW"),
        // "1INCH",
        "000000000000000000000000111111111117dc0aa78b770fa6a738034120c302" => (2, "5wihEYGca7X4gSe97C5mVcqNsfxBzhdTwpv72HKs25US"),
        // "GRT",
        "000000000000000000000000c944e90c64b2c07662a292be6244bdf05cda44a7" => (2, "38i2NQxjp5rt5B3KogqrxmBxgrAwaB3W1f1GmiKqh9MS"),
        // "COMP",
        "000000000000000000000000c00e94cb662c3520282e6f5717214004a7f26888" => (2, "Avz2fmevhhu87WYtWQCFj9UjKRjF9Z9QWwN2ih9yF95G"),
        // "PAXG",
        "00000000000000000000000045804880De22913dAFE09f4980848ECE6EcbAf78" => (2, "9wRD14AhdZ3qV8et3eBQVsrb3UoBZDUbJGyFckpTg8sj"),
};

#[cfg(feature = "test")]
pub static SOLLET_MINTS: phf::Map<&str, (u16, &str)> = phf_map! {
        // "TEST",
        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" => (2, "FDhdMYh3KsF64Jxzh8tnx9rJXQTcN461rguUK9z9zm64"),
};

pub fn derive_mint_for_token(
    program_id: &Pubkey,
    token_address: Address,
    token_chain: ChainID,
) -> (Pubkey, bool) {
    let mut sollet_mint = SOLLET_MINTS.get(hex::encode(token_address).as_str());
    if let Some(mint) = sollet_mint {
        if mint.0 == token_chain {
            return (Pubkey::from_str(mint.1).unwrap(), true);
        }
    }

    (
        WrappedMint::<'_, { AccountState::Uninitialized }>::key(
            &WrappedDerivationData {
                token_chain,
                token_address,
            },
            &program_id,
        ),
        false,
    )
}

pub fn create_wrapped(
    ctx: &ExecutionContext,
    accs: &mut CreateWrapped,
    data: CreateWrappedData,
) -> Result<()> {
    let (mint, is_sollet) =
        derive_mint_for_token(ctx.program_id, accs.vaa.token_address, accs.vaa.token_chain);
    if *accs.mint.info().key != mint {
        return Err(InvalidMint.into());
    }

    let meta_derivation_data: WrappedMetaDerivationData = (&*accs).into();
    accs.meta
        .verify_derivation(ctx.program_id, &meta_derivation_data)?;

    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    accs.vaa.verify(ctx.program_id)?;
    accs.vaa.claim(ctx, accs.payer.key)?;

    if !is_sollet {
        // Create mint account
        accs.mint
            .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

        // Initialize mint
        let init_ix = spl_token::instruction::initialize_mint(
            &spl_token::id(),
            accs.mint.info().key,
            accs.mint_authority.key,
            None,
            min(8, accs.vaa.decimals), // Limit to 8 decimals, truncation is handled on the other side
        )?;
        invoke_signed(&init_ix, ctx.accounts, &[])?;
    }

    // Create meta account
    accs.meta
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

    // Initialize spl meta
    accs.spl_metadata.verify_derivation(
        &spl_token_metadata::id(),
        &SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        },
    )?;

    let mut name = accs.vaa.name.clone();
    name.truncate(32 - 11);
    name += " (Wormhole)";
    let mut symbol = accs.vaa.symbol.clone();
    symbol.truncate(10);

    let spl_token_metadata_ix = spl_token_metadata::instruction::create_metadata_accounts(
        spl_token_metadata::id(),
        *accs.spl_metadata.key,
        *accs.mint.info().key,
        *accs.mint_authority.info().key,
        *accs.payer.info().key,
        *accs.mint_authority.info().key,
        name,
        symbol,
        String::from(""),
        None,
        0,
        false,
        true,
    );
    invoke_seeded(&spl_token_metadata_ix, ctx, &accs.mint_authority, None)?;

    // Populate meta account
    accs.meta.chain = accs.vaa.token_chain;
    accs.meta.token_address = accs.vaa.token_address;
    accs.meta.original_decimals = accs.vaa.decimals;

    Ok(())
}
