use cosmwasm_std::{
    log, to_binary, Api, Binary, CanonicalAddr, CosmosMsg, Env, Extern, HandleResponse, HumanAddr,
    InitResponse, Querier, QueryRequest, StdResult, Storage, Uint128, WasmMsg, WasmQuery,
};

use crate::byte_utils::extend_address_to_32;
use crate::byte_utils::ByteUtils;
use crate::error::ContractError;
use crate::msg::{GuardianSetInfoResponse, HandleMsg, InitMsg, QueryMsg};
use crate::state::{
    config, config_read, guardian_set_get, guardian_set_set, vaa_archive_add, vaa_archive_check,
    wrapped_asset, wrapped_asset_address, wrapped_asset_address_read, wrapped_asset_read,
    ConfigInfo, GuardianAddress, GuardianSetInfo,
};

use cw20_base::msg::HandleMsg as TokenMsg;
use cw20_base::msg::QueryMsg as TokenQuery;

use cw20::TokenInfoResponse;

use cw20_wrapped::msg::HandleMsg as WrappedMsg;
use cw20_wrapped::msg::InitMsg as WrappedInit;
use cw20_wrapped::msg::QueryMsg as WrappedQuery;
use cw20_wrapped::msg::{InitHook, InitMint, WrappedAssetInfoResponse};

use k256::ecdsa::recoverable::Id as RecoverableId;
use k256::ecdsa::recoverable::Signature as RecoverableSignature;
use k256::ecdsa::Signature;
use k256::ecdsa::VerifyKey;
use k256::EncodedPoint;
use sha3::{Digest, Keccak256};

use std::convert::TryFrom;

// Chain ID of Terra
const CHAIN_ID: u8 = 3;

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {
    // Save general wormhole info
    let state = ConfigInfo {
        guardian_set_index: 0,
        guardian_set_expirity: msg.guardian_set_expirity,
        wrapped_asset_code_id: msg.wrapped_asset_code_id,
        owner: deps.api.canonical_address(&env.message.sender)?,
        is_active: true,
    };
    config(&mut deps.storage).save(&state)?;

    // Add initial guardian set to storage
    guardian_set_set(
        &mut deps.storage,
        state.guardian_set_index,
        &msg.initial_guardian_set,
    )?;

    Ok(InitResponse::default())
}

pub fn handle<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: HandleMsg,
) -> StdResult<HandleResponse> {
    match msg {
        HandleMsg::SubmitVAA { vaa } => handle_submit_vaa(deps, env, &vaa.as_slice()),
        HandleMsg::RegisterAssetHook { asset_id } => {
            handle_register_asset(deps, env, &asset_id.as_slice())
        }
        HandleMsg::LockAssets {
            asset,
            recipient,
            amount,
            target_chain,
            nonce,
        } => handle_lock_assets(
            deps,
            env,
            asset,
            amount,
            recipient.as_slice(),
            target_chain,
            nonce,
        ),
        HandleMsg::SetActive { is_active } => handle_set_active(deps, env, is_active),
    }
}

/// Process VAA message signed by quardians
fn handle_submit_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &[u8],
) -> StdResult<HandleResponse> {
    let state = config_read(&deps.storage).load()?;
    if !state.is_active {
        return ContractError::ContractInactive.std_err();
    }

    /* VAA format:

    header (length 6):
    0   uint8   version (0x01)
    1   uint32  guardian set index
    5   uint8   len signatures

    per signature (length 66):
    0   uint8       index of the signer (in guardian keys)
    1   [65]uint8   signature

    body:
    0   uint32  unix seconds
    4   uint8   action
    5   [payload_size]uint8 payload */

    const HEADER_LEN: usize = 6;
    const SIGNATURE_LEN: usize = 66;

    let version = data.get_u8(0);
    if version != 1 {
        return ContractError::InvalidVersion.std_err();
    }

    // Load 4 bytes starting from index 1
    let vaa_guardian_set_index: u32 = data.get_u32(1);
    let len_signers = data.get_u8(5) as usize;
    let body_offset: usize = 6 + SIGNATURE_LEN * len_signers as usize;

    // Hash the body
    let body = &data[body_offset..];
    let mut hasher = Keccak256::new();
    hasher.update(body);
    let hash = hasher.finalize();

    // Check if VAA with this hash was already accepted
    if vaa_archive_check(&deps.storage, &hash) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }

    // Load and check guardian set
    let guardian_set = guardian_set_get(&deps.storage, vaa_guardian_set_index);
    let guardian_set: GuardianSetInfo =
        guardian_set.or_else(|_| ContractError::InvalidGuardianSetIndex.std_err())?;

    if guardian_set.expiration_time != 0 && guardian_set.expiration_time < env.block.time {
        return ContractError::GuardianSetExpired.std_err();
    }
    if len_signers < guardian_set.quorum() {
        return ContractError::NoQuorum.std_err();
    }

    // Verify guardian signatures
    let mut last_index: i32 = -1;
    let mut pos = HEADER_LEN;
    for _ in 0..len_signers {
        let index = data.get_u8(pos) as i32;
        if index <= last_index {
            return ContractError::WrongGuardianIndexOrder.std_err();
        }
        last_index = index;

        let signature = Signature::try_from(&data[pos + 1..pos + 1 + 64])
            .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;
        let id = RecoverableId::new(data.get_u8(pos + 1 + 64))
            .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;
        let recoverable_signature = RecoverableSignature::new(&signature, id)
            .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;

        let verify_key = recoverable_signature
            .recover_verify_key_from_digest_bytes(&hash)
            .or_else(|_| ContractError::CannotRecoverKey.std_err())?;

        let index = index as usize;
        if index >= guardian_set.addresses.len() {
            return ContractError::TooManySignatures.std_err();
        }
        if !keys_equal(&verify_key, &guardian_set.addresses[index]) {
            return ContractError::GuardianSignatureError.std_err();
        }
        pos += SIGNATURE_LEN;
    }

    // Signatures valid, apply VAA
    let action = data.get_u8(body_offset + 4);
    let payload = &data[body_offset + 5..];

    let result = match action {
        0x01 => {
            if vaa_guardian_set_index != state.guardian_set_index {
                return ContractError::NotCurrentGuardianSet.std_err();
            }
            vaa_update_guardian_set(deps, env, payload)
        }
        0x10 => vaa_transfer(deps, env, payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    };

    if result.is_ok() {
        vaa_archive_add(&mut deps.storage, &hash)?;
    }

    result
}

/// Handle wrapped asset registration messages
fn handle_register_asset<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    asset_id: &[u8],
) -> StdResult<HandleResponse> {
    let mut bucket = wrapped_asset(&mut deps.storage);
    let result = bucket.load(asset_id);
    match result {
        Ok(_) => {
            // Asset already registered, return error
            ContractError::AssetAlreadyRegistered.std_err()
        }
        Err(_) => {
            bucket.save(asset_id, &env.message.sender)?;

            let contract_address: CanonicalAddr =
                deps.api.canonical_address(&env.message.sender)?;
            wrapped_asset_address(&mut deps.storage)
                .save(contract_address.as_slice(), &asset_id.to_vec())?;

            Ok(HandleResponse {
                messages: vec![],
                log: vec![
                    log("action", "register_asset"),
                    log("asset_id", format!("{:?}", asset_id)),
                    log("contract_addr", env.message.sender),
                ],
                data: None,
            })
        }
    }
}

fn vaa_update_guardian_set<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &[u8],
) -> StdResult<HandleResponse> {
    /* Payload format
    0   uint32 new_index
    4   uint8 len(keys)
    5   [][20]uint8 guardian addresses
    */

    let mut state = config_read(&deps.storage).load()?;

    let new_guardian_set_index = data.get_u32(0);

    if new_guardian_set_index != state.guardian_set_index + 1 {
        return ContractError::GuardianSetIndexIncreaseError.std_err();
    }
    let len = data.get_u8(4);

    let mut new_guardian_set = GuardianSetInfo {
        addresses: vec![],
        expiration_time: 0,
    };
    let mut pos = 5;
    for _ in 0..len {
        new_guardian_set.addresses.push(GuardianAddress {
            bytes: data[pos..pos + 20].to_vec().into(),
        });
        pos += 20;
    }

    let old_guardian_set_index = state.guardian_set_index;
    state.guardian_set_index = new_guardian_set_index;

    guardian_set_set(
        &mut deps.storage,
        state.guardian_set_index,
        &new_guardian_set,
    )?;
    config(&mut deps.storage).save(&state)?;

    let mut old_guardian_set = guardian_set_get(&deps.storage, old_guardian_set_index)?;
    old_guardian_set.expiration_time = env.block.time + state.guardian_set_expirity;
    guardian_set_set(&mut deps.storage, old_guardian_set_index, &old_guardian_set)?;

    // TODO: Apply new guardian set
    Ok(HandleResponse {
        messages: vec![],
        log: vec![
            log("action", "guardian_set_change"),
            log("old", old_guardian_set_index),
            log("new", state.guardian_set_index),
        ],
        data: None,
    })
}

fn vaa_transfer<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &[u8],
) -> StdResult<HandleResponse> {
    /* Payload format:
    0   uint32 nonce
    4   uint8 source_chain
    5   uint8 target_chain
    6   [32]uint8 source_address
    38  [32]uint8 target_address
    70  uint8 token_chain
    71  [32]uint8 token_address
    103 uint8 decimals
    104 uint256 amount */

    let source_chain = data.get_u8(4);
    let target_chain = data.get_u8(5);

    let target_address = data.get_address(38);

    let token_chain = data.get_u8(70);
    let (not_supported_amount, amount) = data.get_u256(104);

    // Check high 128 bit of amount value to be empty
    if not_supported_amount != 0 {
        return ContractError::AmountTooHigh.std_err();
    }

    // Check if source and target chains are different
    if source_chain == target_chain {
        return ContractError::SameSourceAndTarget.std_err();
    }

    // Check if transfer is incoming
    if target_chain != CHAIN_ID {
        return ContractError::WrongTargetChain.std_err();
    }

    if token_chain != CHAIN_ID {
        let mut asset_id: Vec<u8> = vec![];
        asset_id.push(token_chain);
        let asset_address = data.get_bytes32(71);
        asset_id.extend_from_slice(asset_address);

        let mut hasher = Keccak256::new();
        hasher.update(asset_id);
        let asset_id = hasher.finalize();

        let mut messages: Vec<CosmosMsg> = vec![];

        // Check if this asset is already deployed
        match wrapped_asset_read(&deps.storage).load(&asset_id) {
            Ok(contract_addr) => {
                // Asset already deployed, just mint
                messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr,
                    msg: to_binary(&WrappedMsg::Mint {
                        recipient: deps
                            .api
                            .human_address(&target_address)
                            .or_else(|_| ContractError::WrongTargetAddressFormat.std_err())?,
                        amount: Uint128::from(amount),
                    })?,
                    send: vec![],
                }));
            }
            Err(_) => {
                // Asset is not deployed yet, deploy and mint
                let state = config_read(&deps.storage).load()?;
                messages.push(CosmosMsg::Wasm(WasmMsg::Instantiate {
                    code_id: state.wrapped_asset_code_id,
                    msg: to_binary(&WrappedInit {
                        asset_chain: token_chain,
                        asset_address: asset_address.to_vec().into(),
                        decimals: data.get_u8(103),
                        mint: Some(InitMint {
                            recipient: deps
                                .api
                                .human_address(&target_address)
                                .or_else(|_| ContractError::WrongTargetAddressFormat.std_err())?,
                            amount: Uint128::from(amount),
                        }),
                        init_hook: Some(InitHook {
                            contract_addr: env.contract.address,
                            msg: to_binary(&HandleMsg::RegisterAssetHook {
                                asset_id: asset_id.to_vec().into(),
                            })?,
                        }),
                    })?,
                    send: vec![],
                    label: None,
                }));
            }
        }

        Ok(HandleResponse {
            messages,
            log: vec![],
            data: None,
        })
    } else {
        let token_address = data.get_address(71);

        Ok(HandleResponse {
            messages: vec![CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: deps.api.human_address(&token_address)?,
                msg: to_binary(&TokenMsg::Transfer {
                    recipient: deps.api.human_address(&target_address)?,
                    amount: Uint128::from(amount),
                })?,
                send: vec![],
            })],
            log: vec![], // TODO: Add log entries
            data: None,
        })
    }
}

fn handle_lock_assets<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    asset: HumanAddr,
    amount: Uint128,
    recipient: &[u8],
    target_chain: u8,
    nonce: u32,
) -> StdResult<HandleResponse> {
    if target_chain == CHAIN_ID {
        return ContractError::SameSourceAndTarget.std_err();
    }

    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }

    let state = config_read(&deps.storage).load()?;
    if !state.is_active {
        return ContractError::ContractInactive.std_err();
    }

    let asset_chain: u8;
    let asset_address: Vec<u8>;

    // Query asset details
    let request = QueryRequest::<()>::Wasm(WasmQuery::Smart {
        contract_addr: asset.clone(),
        msg: to_binary(&TokenQuery::TokenInfo {})?,
    });
    let token_info: TokenInfoResponse = deps.querier.custom_query(&request)?;

    let decimals: u8 = token_info.decimals;

    let asset_canonical: CanonicalAddr = deps.api.canonical_address(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];

    match wrapped_asset_address_read(&deps.storage).load(asset_canonical.as_slice()) {
        Ok(_) => {
            // This is a deployed wrapped asset, burn it
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: asset.clone(),
                msg: to_binary(&WrappedMsg::Burn {
                    account: env.message.sender.clone(),
                    amount,
                })?,
                send: vec![],
            }));
            let request = QueryRequest::<()>::Wasm(WasmQuery::Smart {
                contract_addr: asset,
                msg: to_binary(&WrappedQuery::WrappedAssetInfo {})?,
            });
            let wrapped_token_info: WrappedAssetInfoResponse =
                deps.querier.custom_query(&request)?;
            asset_chain = wrapped_token_info.asset_chain;
            asset_address = wrapped_token_info.asset_address.as_slice().to_vec();
        }
        Err(_) => {
            // This is a regular asset, transfer its balance
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: asset,
                msg: to_binary(&TokenMsg::TransferFrom {
                    owner: env.message.sender.clone(),
                    recipient: env.contract.address.clone(),
                    amount,
                })?,
                send: vec![],
            }));
            asset_address = extend_address_to_32(&asset_canonical);
            asset_chain = CHAIN_ID;
        }
    };

    Ok(HandleResponse {
        messages,
        log: vec![
            log("locked.target_chain", target_chain),
            log("locked.token_chain", asset_chain),
            log("locked.token_decimals", decimals),
            log("locked.token", hex::encode(asset_address)),
            log(
                "locked.sender",
                hex::encode(extend_address_to_32(
                    &deps.api.canonical_address(&env.message.sender)?,
                )),
            ),
            log("locked.recipient", hex::encode(recipient)),
            log("locked.amount", amount),
            log("locked.nonce", nonce),
            log("locked.block_time", env.block.time),
        ],
        data: None,
    })
}

pub fn handle_set_active<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    is_active: bool,
) -> StdResult<HandleResponse> {
    let mut state = config_read(&deps.storage).load()?;

    if deps.api.canonical_address(&env.message.sender)? != state.owner {
        return ContractError::PermissionDenied.std_err();
    }

    state.is_active = is_active;

    config(&mut deps.storage).save(&state)?;

    Ok(HandleResponse {
        messages: vec![],
        log: vec![],
        data: None,
    })
}

pub fn query<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
        QueryMsg::GuardianSetInfo {} => to_binary(&query_query_guardian_set_info(deps)?),
    }
}

pub fn query_query_guardian_set_info<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
) -> StdResult<GuardianSetInfoResponse> {
    let state = config_read(&deps.storage).load()?;
    let guardian_set = guardian_set_get(&deps.storage, state.guardian_set_index)?;
    let res = GuardianSetInfoResponse {
        guardian_set_index: state.guardian_set_index,
        addresses: guardian_set.addresses,
    };
    Ok(res)
}

fn keys_equal(a: &VerifyKey, b: &GuardianAddress) -> bool {
    let mut hasher = Keccak256::new();

    let point: EncodedPoint = EncodedPoint::from(a);
    let point = point.decompress();
    if bool::from(point.is_none()) {
        return false;
    }
    let point = point.unwrap();

    hasher.update(&point.as_bytes()[1..]);
    let a = &hasher.finalize()[12..];

    let b = &b.bytes;
    if a.len() != b.len() {
        return false;
    }
    for (ai, bi) in a.iter().zip(b.as_slice().iter()) {
        if ai != bi {
            return false;
        }
    }
    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::GuardianSetInfo;
    use cosmwasm_std::testing::{mock_dependencies, mock_env};
    use cosmwasm_std::HumanAddr;
    use std::time::{SystemTime, UNIX_EPOCH};

    // Constants generated by bridge/cmd/vaa-test-terra/main.go
    const ADDR_1: &str = "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe";
    const ADDR_2: &str = "E06A9ADfeB38a8eE4D00E89307C016D0749679bD";
    const ADDR_3: &str = "8575Df9b3c97B4E267Deb92d93137844A97A0132";
    const ADDR_4: &str = "0427cDA59902Dc6EB0c1bd2b6D38F87c5552b348";
    const ADDR_5: &str = "bFEa822F75c42e1764c791B8fE04a7B10DDB3857";
    const ADDR_6: &str = "2F5FE0B158147e7260f14062556AfC94Eece55fF";
    const VAA_VALID_TRANSFER_1_SIG: &str = "01000000000100d106d4f363c6e3d0bf8ebf3cf8ef1ba35e66687b7613a826b5f5b68e0c346e1e0fdd6ceb332c87dad7d170ee6736571c0b75173787a8dcf41a492075e18a9a9601000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_TRANSFER_2_SIGS: &str = "0100000000020040d91705d211c52c9f120adb1b794355ba10ec1ff855295e677c5b341b2e5449684179f8ca4087e88de2cba0e6cbf6e0c7a353529800ccf96e5fdd80a85a59220001efb8a4825c87ab68190e1b184eeda5c45f82b22450ff113f2581a2f1bd3aeca60798392405cd4d3b523a5c3426d09b963c195c842a0040e93651cb700785d0e600000007d0100000003801030201040000000000000000000000000000000000000000000000000000000000000000000000000000000000010203040506070809000102030405060708090002000000000000000000000000d833215cbcc3f914bd1c9ece3ee7bf8b14f841bb080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_TRANSFER_3_SIGS: &str = "0100000000030040d91705d211c52c9f120adb1b794355ba10ec1ff855295e677c5b341b2e5449684179f8ca4087e88de2cba0e6cbf6e0c7a353529800ccf96e5fdd80a85a59220001efb8a4825c87ab68190e1b184eeda5c45f82b22450ff113f2581a2f1bd3aeca60798392405cd4d3b523a5c3426d09b963c195c842a0040e93651cb700785d0e60002a5fb92ff2b5a5eed98e2909ed932e5d9328cb2527027cce8f40c4f5677c341c83fe9fac7bf39af60fe47ecfb6f52b22b9d817d24d4147684b08e2fe19ff3a3ef01000007d0100000003801030201040000000000000000000000000000000000000000000000000000000000000000000000000000000000010203040506070809000102030405060708090002000000000000000000000000d833215cbcc3f914bd1c9ece3ee7bf8b14f841bb080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_GUARDIAN_SET_CHANGE_FROM_0: &str = "01000000000100a33c022217ccb87a5bc83b71e6377fff6639e7904d9e9995a42dc0867dc2b0bc5d1aacc3752ea71cf4d85278526b5dd40b0343667a2d4434a44cbf7844181a1000000007d0010000000101e06a9adfeb38a8ee4d00e89307c016d0749679bd";
    const VAA_ERROR_SIGNATURE_SEQUENCE: &str = "01000000000201efb8a4825c87ab68190e1b184eeda5c45f82b22450ff113f2581a2f1bd3aeca60798392405cd4d3b523a5c3426d09b963c195c842a0040e93651cb700785d0e6000040d91705d211c52c9f120adb1b794355ba10ec1ff855295e677c5b341b2e5449684179f8ca4087e88de2cba0e6cbf6e0c7a353529800ccf96e5fdd80a85a592200000007d0100000003801030201040000000000000000000000000000000000000000000000000000000000000000000000000000000000010203040506070809000102030405060708090002000000000000000000000000d833215cbcc3f914bd1c9ece3ee7bf8b14f841bb080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_ERROR_WRONG_SIGNATURE_1_SIG: &str = "0100000000010075c1b20fb59adc55a08f9778bc525507a36a29d1f0e2cb3fcc9c90f7331786263c4bd53ce5d3865b4f63cddeafb2c1026b5e13f1b66af7dabbd1f1af9f34fd3f01000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_GUARDIAN_SET_CHANGE_FROM_0_DIFF: &str = "01000000000100d90d6f9cbc0458599cbe4d267bc9221b54955b94cb5cb338aeb845bdc9dd275f558871ea479de9cc0b44cfb2a07344431a3adbd2f98aa86f4e12ff4aba061b7f00000007d00100000001018575df9b3c97b4e267deb92d93137844a97a0132";
    const VAA_ERROR_INVALID_TARGET_ADDRESS: &str = "0100000000010092f32c76aa3a8d83de59b3f2281cfbf70af33d9bcfbaa78bd3e9cafc512335ab40b126a894f0182ee8c69f5324496eb681c1780ed39bcc80f589cfc0a5df144a01000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_GUARDIAN_SET_CHANGE_JUMP: &str = "010000000001004b179853b36b76446c72944d50551be814ab34f23da2124615315da71505df801b38355d741cdd65e856792e2a1435270abfe52ae005c4e3671c0b7aac36445a01000007d00100000002018575df9b3c97b4e267deb92d93137844a97a0132";
    const VAA_ERROR_AMOUNT_TOO_HIGH: &str = "0100000000010055fdf76a64b779ac5b7a54dc181cf430f4d14a499b7933049d8bc94db529ed0a2d12d50ec2026883e59a5c64f2189b60c84a53b66113e8b52da66fd89f70495f00000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000100000000000000000000000000000000";
    const VAA_ERROR_SAME_SOURCE_AND_TARGET: &str = "010000000001004c53dfce8fc9e781f0cfdc6592c00c337c1e109168ff17ee3bf4cf69ddb8a0a52a3c215093301d5459d282d625dc5125592609f06f14a57f61121e668b0ec10500000003e81000000038030302010400000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_ERROR_WRONG_TARGET: &str = "01000000000100b19a265b1407e9619ffc29be9562161ed2c155db5ba68e01265a250a677eb0c62bb91e468da827e9ec4c1e9428ade97129126f56500c4a3c9f9803cc85f656d200000003e81000000038010202010400000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_GUARDIAN_SET_CHANGE_TO_6: &str = "01000000000100a5defbd912ef327d07afff71e0da9c2e2a13e5516255c62e249a6761afe2465c7b6fc1032451559551e76eb4a029474fd791b2250c4fd40a8b3f5d4f5f58e5a30000000fa0010000000106befa429d57cd18b7f8a4d91a2da9ab4af05d0fbee06a9adfeb38a8ee4d00e89307c016d0749679bd8575df9b3c97b4e267deb92d93137844a97a01320427cda59902dc6eb0c1bd2b6d38f87c5552b348bfea822f75c42e1764c791b8fe04a7b10ddb38572f5fe0b158147e7260f14062556afc94eece55ff";
    const VAA_VALID_TRANSFER_5_SIGS_GS_1: &str = "01000000010500027eb7e87a9d0ab91ec53bb073c0f0acf189900139daa652666fd4cfe32a4ee42383c1a66e3a397c2de8ae485225357feb52f665952b1e384ef6dfcea1ba9f920001cfcacfad444ac3202f8f0d2252c69ee90d18c9105f7be3b5d361b7fcb0fbf7fa7287bac5de9cb02f86a28fdd7f24015991020431b0048aa3bbb29daed625e416000372f6c239ddeccded04a95a0cf0bfefe6e168148f1fe3b93e797eb2e74e098b890f2be341dd0f3c8172c2050154407cfdd1ea7bd6cce0b31f020ec7530ffb6109000449c025fe0630268983d57c4bd1546497788f810e427b6fd436cb1f048152375e1063422b4d1cc668a0612814c550ea7e3d1aa93404a0b6e089d210d4c937023a000548bf474fb350d5e482378c37404fb4d1421e262d13ebf6b11977214c789a246a6c278a522a9be4beba008f3d481b1ee35c5b0559bef474eb34b9e3e681947c230100000fa01000000039010302010500000000000000000000000000000000000000000000000000000000000000000000000000000000000102030405060708090001020304050607080900010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";

    const CANONICAL_LENGTH: usize = 20;

    lazy_static! {
        static ref ALL_GUARDIANS: Vec<GuardianAddress> = vec![
            GuardianAddress::from(ADDR_1),
            GuardianAddress::from(ADDR_2),
            GuardianAddress::from(ADDR_3),
            GuardianAddress::from(ADDR_4),
            GuardianAddress::from(ADDR_5),
            GuardianAddress::from(ADDR_6)
        ];
    }

    fn unix_timestamp() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs()
    }

    fn do_init_with_guardians<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        number_of_guardians: usize,
    ) {
        let expiration_time = unix_timestamp() + 1000;
        do_init(deps, &ALL_GUARDIANS[..number_of_guardians], expiration_time);
    }

    fn do_init<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        guardians: &[GuardianAddress],
        expiration_time: u64,
    ) {
        let init_msg = InitMsg {
            initial_guardian_set: GuardianSetInfo {
                addresses: guardians.to_vec(),
                expiration_time,
            },
            guardian_set_expirity: 50,
            wrapped_asset_code_id: 999,
        };
        let mut env = mock_env(&HumanAddr::from("creator"), &[]);
        env.block.time = unix_timestamp();
        let res = init(deps, env, init_msg).unwrap();
        assert_eq!(0, res.messages.len());

        // TODO: Query and check contract state and guardians storage
    }

    fn submit_vaa<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        vaa: &str,
    ) -> StdResult<HandleResponse> {
        let msg = HandleMsg::SubmitVAA {
            vaa: hex::decode(vaa).expect("Decoding failed").into(),
        };
        let mut env = mock_env(&HumanAddr::from("creator"), &[]);
        env.block.time = unix_timestamp();

        handle(deps, env, msg)
    }

    #[test]
    fn can_init() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);
    }

    #[test]
    fn valid_vaa_token_transfer() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let messages = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG)
            .unwrap()
            .messages;
        assert_eq!(1, messages.len());
        let msg = &messages[0];
        match msg {
            CosmosMsg::Wasm(wasm_msg) => match wasm_msg {
                WasmMsg::Instantiate {
                    code_id,
                    msg: _,
                    send,
                    label,
                } => {
                    assert_eq!(*code_id, 999);
                    assert_eq!(*label, None);
                    assert_eq!(*send, vec![]);
                }
                _ => panic!("Wrong message type"),
            },
            _ => panic!("Wrong message type"),
        }
    }

    #[test]
    fn valid_vaa_2_signatures() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 2);

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_2_SIGS);
        assert!(result.is_ok());
    }

    #[test]
    fn valid_vaa_non_expiring_guardians() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init(&mut deps, &vec![GuardianAddress::from(ADDR_1)], 0);

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG);
        assert!(result.is_ok());
    }

    #[test]
    fn error_vaa_same_vaa_twice() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let _ = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG).unwrap();
        let e = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG).unwrap_err();
        assert_eq!(e, ContractError::VaaAlreadyExecuted.std());
    }

    #[test]
    fn valid_vaa_guardian_set_change() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let messages = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE_FROM_0)
            .unwrap()
            .messages;
        assert_eq!(0, messages.len());

        // Check storage
        let state = config_read(&deps.storage)
            .load()
            .expect("Cannot load config storage");
        assert_eq!(state.guardian_set_index, 1);
        let guardian_set_info = guardian_set_get(&deps.storage, state.guardian_set_index)
            .expect("Cannot find guardian set");
        assert_eq!(
            guardian_set_info,
            GuardianSetInfo {
                addresses: vec![GuardianAddress::from(ADDR_2)],
                expiration_time: 0
            }
        );
    }

    #[test]
    fn error_vaa_guardian_set_expired() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        // Expiration time 1 second in the past
        let expiration_time = unix_timestamp() - 1;
        do_init(
            &mut deps,
            &vec![GuardianAddress::from(ADDR_1)],
            expiration_time,
        );

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG);
        assert_eq!(result, ContractError::GuardianSetExpired.std_err());
    }

    #[test]
    fn error_vaa_no_quorum() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 2);

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_1_SIG);
        assert_eq!(result, ContractError::NoQuorum.std_err());
    }

    #[test]
    fn valid_partial_quorum() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 4);

        // 3 signatures on 4-guardian set is quorum
        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_3_SIGS);
        assert!(result.is_ok());
    }

    #[test]
    fn error_vaa_wrong_guardian_index_order() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 2);

        let result = submit_vaa(&mut deps, VAA_ERROR_SIGNATURE_SEQUENCE);
        assert_eq!(result, ContractError::WrongGuardianIndexOrder.std_err());
    }

    #[test]
    fn error_vaa_too_many_signatures() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_2_SIGS);
        assert_eq!(result, ContractError::TooManySignatures.std_err());
    }

    #[test]
    fn error_vaa_invalid_signature() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init(
            &mut deps,
            // Use 1-2-4 guardians
            &vec![
                GuardianAddress::from(ADDR_1),
                GuardianAddress::from(ADDR_2),
                GuardianAddress::from(ADDR_4),
            ],
            unix_timestamp(),
        );
        // Sign by 1-2-3 guardians
        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_3_SIGS);
        assert_eq!(result, ContractError::GuardianSignatureError.std_err());

        // Single signature, wrong key
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_ERROR_WRONG_SIGNATURE_1_SIG);
        assert_eq!(result, ContractError::GuardianSignatureError.std_err());
    }

    #[test]
    fn error_vaa_not_current_quardian_set() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE_FROM_0);
        assert!(result.is_ok());

        // Submit another valid change, which will fail, because now set #1 is active
        // (we need to send a different VAA, because otherwise it will be blocked by duplicate check)
        let result = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE_FROM_0_DIFF);
        assert_eq!(result, ContractError::NotCurrentGuardianSet.std_err());
    }

    #[test]
    fn error_vaa_wrong_target_address_format() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_ERROR_INVALID_TARGET_ADDRESS);
        assert_eq!(result, ContractError::WrongTargetAddressFormat.std_err());
    }

    #[test]
    fn error_vaa_guardian_set_change_index_not_increasing() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE_JUMP);
        assert_eq!(
            result,
            ContractError::GuardianSetIndexIncreaseError.std_err()
        );
    }

    #[test]
    fn error_vaa_transfer_amount_too_high() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_ERROR_AMOUNT_TOO_HIGH);
        assert_eq!(result, ContractError::AmountTooHigh.std_err());
    }

    #[test]
    fn error_vaa_transfer_same_source_and_target() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_ERROR_SAME_SOURCE_AND_TARGET);
        assert_eq!(result, ContractError::SameSourceAndTarget.std_err());
    }

    #[test]
    fn error_vaa_transfer_wrong_target_chain() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_ERROR_WRONG_TARGET);
        assert_eq!(result, ContractError::WrongTargetChain.std_err());
    }

    #[test]
    fn valid_transfer_after_guardian_set_change() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_with_guardians(&mut deps, 1);

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_5_SIGS_GS_1);
        assert_eq!(result, ContractError::InvalidGuardianSetIndex.std_err());

        let result = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE_TO_6);
        assert!(result.is_ok());

        let result = submit_vaa(&mut deps, VAA_VALID_TRANSFER_5_SIGS_GS_1);
        assert!(result.is_ok());
    }
}
