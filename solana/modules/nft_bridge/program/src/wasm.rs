use crate::{
    accounts::{
        AuthoritySigner,
        EmitterAccount,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    instructions::{
        complete_native,
        complete_wrapped,
        complete_wrapped_meta,
        register_chain,
        transfer_native,
        transfer_wrapped,
        upgrade_contract,
    },
    messages::{
        GovernancePayloadUpgrade,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    },
    types::{
        EndpointRegistration,
        WrappedMeta,
    },
    CompleteNativeData,
    CompleteWrappedData,
    CompleteWrappedMetaData,
    RegisterChainData,
    TransferNativeData,
    TransferWrappedData,
};
use borsh::BorshDeserialize;
use bridge::{
    accounts::PostedVAADerivationData,
    instructions::hash_vaa,
    vaa::VAA,
    DeserializePayload,
    PostVAAData,
};
use primitive_types::U256;
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use std::str::FromStr;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn transfer_native_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    message: String,
    from: String,
    mint: String,
    nonce: u32,
    target_address: Vec<u8>,
    target_chain: u16,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let message = Pubkey::from_str(message.as_str()).unwrap();
    let from = Pubkey::from_str(from.as_str()).unwrap();
    let mint = Pubkey::from_str(mint.as_str()).unwrap();

    let mut target_addr = [0u8; 32];
    target_addr.copy_from_slice(target_address.as_slice());

    let ix = transfer_native(
        program_id,
        bridge_id,
        payer,
        message,
        from,
        mint,
        TransferNativeData {
            nonce,
            target_address: target_addr,
            target_chain,
        },
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn transfer_wrapped_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    message: String,
    from: String,
    from_owner: String,
    token_chain: u16,
    token_address: Vec<u8>,
    token_id: Vec<u8>,
    nonce: u32,
    target_address: Vec<u8>,
    target_chain: u16,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let message = Pubkey::from_str(message.as_str()).unwrap();
    let from = Pubkey::from_str(from.as_str()).unwrap();
    let from_owner = Pubkey::from_str(from_owner.as_str()).unwrap();

    let mut target_addr = [0u8; 32];
    target_addr.copy_from_slice(target_address.as_slice());
    let mut token_addr = [0u8; 32];
    token_addr.copy_from_slice(token_address.as_slice());
    let token_id = U256::from_big_endian(token_id.as_slice());

    let ix = transfer_wrapped(
        program_id,
        bridge_id,
        payer,
        message,
        from,
        from_owner,
        token_chain,
        token_addr,
        token_id,
        TransferWrappedData {
            nonce,
            target_address: target_addr,
            target_chain,
        },
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn complete_transfer_native_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    to_authority: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let to_authority = Pubkey::from_str(to_authority.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = PayloadTransfer::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = bridge::accounts::PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &bridge_id,
    );
    let post_vaa_data = PostVAAData {
        version: vaa.version,
        guardian_set_index: vaa.guardian_set_index,
        timestamp: vaa.timestamp,
        nonce: vaa.nonce,
        emitter_chain: vaa.emitter_chain,
        emitter_address: vaa.emitter_address,
        sequence: vaa.sequence,
        consistency_level: vaa.consistency_level,
        payload: vaa.payload,
    };

    let mut mint_bytes = [0u8; 32];
    payload.token_id.to_big_endian(&mut mint_bytes);
    let ix = complete_native(
        program_id,
        bridge_id,
        payer,
        message_key,
        post_vaa_data,
        to_authority,
        Pubkey::new(&mint_bytes),
        CompleteNativeData {},
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn complete_transfer_wrapped_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    to_authority: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let to_authority = Pubkey::from_str(to_authority.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = PayloadTransfer::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = bridge::accounts::PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &bridge_id,
    );
    let post_vaa_data = PostVAAData {
        version: vaa.version,
        guardian_set_index: vaa.guardian_set_index,
        timestamp: vaa.timestamp,
        nonce: vaa.nonce,
        emitter_chain: vaa.emitter_chain,
        emitter_address: vaa.emitter_address,
        sequence: vaa.sequence,
        consistency_level: vaa.consistency_level,
        payload: vaa.payload,
    };

    let ix = complete_wrapped(
        program_id,
        bridge_id,
        payer,
        message_key,
        post_vaa_data,
        payload.clone(),
        to_authority,
        CompleteWrappedData {},
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn complete_transfer_wrapped_meta_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = PayloadTransfer::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = bridge::accounts::PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &bridge_id,
    );
    let post_vaa_data = PostVAAData {
        version: vaa.version,
        guardian_set_index: vaa.guardian_set_index,
        timestamp: vaa.timestamp,
        nonce: vaa.nonce,
        emitter_chain: vaa.emitter_chain,
        emitter_address: vaa.emitter_address,
        sequence: vaa.sequence,
        consistency_level: vaa.consistency_level,
        payload: vaa.payload,
    };

    let ix = complete_wrapped_meta(
        program_id,
        bridge_id,
        payer,
        message_key,
        post_vaa_data,
        payload.clone(),
        CompleteWrappedMetaData {},
    )
    .unwrap();

    JsValue::from_serde(&ix).unwrap()
}

#[wasm_bindgen]
pub fn upgrade_contract_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    spill: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let spill = Pubkey::from_str(spill.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = GovernancePayloadUpgrade::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = bridge::accounts::PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &bridge_id,
    );
    let ix = upgrade_contract(
        program_id,
        Pubkey::from_str(payer.as_str()).unwrap(),
        message_key,
        Pubkey::new(&vaa.emitter_address),
        payload.new_contract,
        spill,
        vaa.sequence,
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn register_chain_ix(
    program_id: String,
    bridge_id: String,
    payer: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let bridge_id = Pubkey::from_str(bridge_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = PayloadGovernanceRegisterChain::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = bridge::accounts::PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &bridge_id,
    );
    let post_vaa_data = PostVAAData {
        version: vaa.version,
        guardian_set_index: vaa.guardian_set_index,
        timestamp: vaa.timestamp,
        nonce: vaa.nonce,
        emitter_chain: vaa.emitter_chain,
        emitter_address: vaa.emitter_address,
        sequence: vaa.sequence,
        consistency_level: vaa.consistency_level,
        payload: vaa.payload,
    };
    let ix = register_chain(
        program_id,
        bridge_id,
        payer,
        message_key,
        post_vaa_data,
        payload,
        RegisterChainData {},
    )
    .unwrap();
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn emitter_address(program_id: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let emitter = EmitterAccount::key(None, &program_id);

    emitter.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn approval_authority_address(program_id: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let approval_authority = AuthoritySigner::key(None, &program_id);

    approval_authority.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn wrapped_address(
    program_id: String,
    token_address: Vec<u8>,
    token_chain: u16,
    token_id: Vec<u8>,
) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let mut t_addr = [0u8; 32];
    t_addr.copy_from_slice(&token_address);
    let token_id = U256::from_big_endian(token_id.as_slice());

    let wrapped_addr = WrappedMint::<'_, { AccountState::Initialized }>::key(
        &WrappedDerivationData {
            token_address: t_addr,
            token_chain,
            token_id,
        },
        &program_id,
    );

    wrapped_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn wrapped_meta_address(program_id: String, mint_address: Vec<u8>) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let mint_key = Pubkey::new(mint_address.as_slice());

    let wrapped_meta_addr = WrappedTokenMeta::<'_, { AccountState::Initialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );

    wrapped_meta_addr.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn spl_meta_address(mint_address: Vec<u8>) -> Vec<u8> {
    let mint_key = Pubkey::new(mint_address.as_slice());

    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData { mint: mint_key },
        &spl_token_metadata::id(),
    );

    spl_metadata.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_wrapped_meta(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&WrappedMeta::try_from_slice(data.as_slice()).unwrap()).unwrap()
}

#[wasm_bindgen]
pub fn parse_endpoint_registration(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&EndpointRegistration::try_from_slice(data.as_slice()).unwrap()).unwrap()
}
