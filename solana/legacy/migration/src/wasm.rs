use crate::{
    accounts::{
        AuthoritySigner,
        FromCustodyTokenAccount,
        FromCustodyTokenAccountDerivationData,
        MigrationPool,
        MigrationPoolDerivationData,
        ShareMint,
        ShareMintDerivationData,
        ToCustodyTokenAccount,
        ToCustodyTokenAccountDerivationData,
    },
    instructions,
    types::PoolData,
};
use borsh::BorshDeserialize;
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use std::str::FromStr;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn add_liquidity(
    program_id: String,
    from_mint: String,
    to_mint: String,
    liquidity_token_account: String,
    lp_share_token_account: String,
    amount: u64,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let from_mint = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint = Pubkey::from_str(to_mint.as_str()).unwrap();
    let liquidity_token_account = Pubkey::from_str(liquidity_token_account.as_str()).unwrap();
    let lp_share_token_account = Pubkey::from_str(lp_share_token_account.as_str()).unwrap();

    let ix = instructions::add_liquidity(
        program_id,
        from_mint,
        to_mint,
        liquidity_token_account,
        lp_share_token_account,
        amount,
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn remove_liquidity(
    program_id: String,
    from_mint: String,
    to_mint: String,
    liquidity_token_account: String,
    lp_share_token_account: String,
    amount: u64,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let from_mint = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint = Pubkey::from_str(to_mint.as_str()).unwrap();
    let liquidity_token_account = Pubkey::from_str(liquidity_token_account.as_str()).unwrap();
    let lp_share_token_account = Pubkey::from_str(lp_share_token_account.as_str()).unwrap();

    let ix = instructions::remove_liquidity(
        program_id,
        from_mint,
        to_mint,
        liquidity_token_account,
        lp_share_token_account,
        amount,
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn claim_shares(
    program_id: String,
    from_mint: String,
    to_mint: String,
    output_token_account: String,
    lp_share_token_account: String,
    amount: u64,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let from_mint = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint = Pubkey::from_str(to_mint.as_str()).unwrap();
    let output_token_account = Pubkey::from_str(output_token_account.as_str()).unwrap();
    let lp_share_token_account = Pubkey::from_str(lp_share_token_account.as_str()).unwrap();

    let ix = instructions::claim_shares(
        program_id,
        from_mint,
        to_mint,
        output_token_account,
        lp_share_token_account,
        amount,
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn create_pool(
    program_id: String,
    payer: String,
    from_mint: String,
    to_mint: String,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let from_mint = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint = Pubkey::from_str(to_mint.as_str()).unwrap();

    let ix = instructions::create_pool(program_id, payer, from_mint, to_mint).unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn migrate_tokens(
    program_id: String,
    from_mint: String,
    to_mint: String,
    input_token_account: String,
    output_token_account: String,
    amount: u64,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let from_mint = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint = Pubkey::from_str(to_mint.as_str()).unwrap();
    let input_token_account = Pubkey::from_str(input_token_account.as_str()).unwrap();
    let output_token_account = Pubkey::from_str(output_token_account.as_str()).unwrap();

    let ix = instructions::migrate_tokens(
        program_id,
        from_mint,
        to_mint,
        input_token_account,
        output_token_account,
        amount,
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn pool_address(program_id: String, from_mint: String, to_mint: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let from_mint_key = Pubkey::from_str(from_mint.as_str()).unwrap();
    let to_mint_key = Pubkey::from_str(to_mint.as_str()).unwrap();

    let pool_addr = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint_key,
            to: to_mint_key,
        },
        &program_id,
    );

    pool_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn authority_address(program_id: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();

    let authority_addr = AuthoritySigner::key(None, &program_id);

    authority_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn share_mint_address(program_id: String, pool: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let pool_key = Pubkey::from_str(pool.as_str()).unwrap();

    let share_mint_addr = ShareMint::<'_, { AccountState::Initialized }>::key(
        &ShareMintDerivationData { pool: pool_key },
        &program_id,
    );

    share_mint_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn from_custody_address(program_id: String, pool: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let pool_key = Pubkey::from_str(pool.as_str()).unwrap();

    let from_custody_addr = FromCustodyTokenAccount::<'_, { AccountState::Initialized }>::key(
        &FromCustodyTokenAccountDerivationData { pool: pool_key },
        &program_id,
    );

    from_custody_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn to_custody_address(program_id: String, pool: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let pool_key = Pubkey::from_str(pool.as_str()).unwrap();

    let to_custody_addr = ToCustodyTokenAccount::<'_, { AccountState::Initialized }>::key(
        &ToCustodyTokenAccountDerivationData { pool: pool_key },
        &program_id,
    );

    to_custody_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_pool(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&PoolData::try_from_slice(data.as_slice()).unwrap()).unwrap()
}
