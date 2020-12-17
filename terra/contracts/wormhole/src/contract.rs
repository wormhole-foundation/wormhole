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

use hex;

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
        HandleMsg::RegisterAssetHook { asset_id } => handle_register_asset(deps, env, &asset_id.as_slice()),
        HandleMsg::LockAssets {
            asset,
            recipient,
            amount,
            target_chain,
            nonce,
        } => handle_lock_assets(deps, env, asset, amount, recipient.as_slice(), target_chain, nonce),
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
        guardian_set.or(ContractError::InvalidGuardianSetIndex.std_err())?;

    if guardian_set.expiration_time == 0 || guardian_set.expiration_time < env.block.time {
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
            return Err(ContractError::WrongGuardianIndexOrder.std());
        }
        last_index = index;

        let signature = Signature::try_from(&data[pos + 1..pos + 1 + 64])
            .or(ContractError::CannotDecodeSignature.std_err())?;
        let id = RecoverableId::new(data.get_u8(pos + 1 + 64))
            .or(ContractError::CannotDecodeSignature.std_err())?;
        let recoverable_signature = RecoverableSignature::new(&signature, id)
            .or(ContractError::CannotDecodeSignature.std_err())?;

        let verify_key = recoverable_signature
            .recover_verify_key_from_digest_bytes(&hash)
            .or(ContractError::CannotRecoverKey.std_err())?;
        if !keys_equal(&verify_key, &guardian_set.addresses[index as usize]) {
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
            return ContractError::AssetAlreadyRegistered.std_err();
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
                        recipient: deps.api.human_address(&target_address)?,
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
                            recipient: deps.api.human_address(&target_address)?,
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
                    amount: Uint128::from(amount),
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
                contract_addr: asset.clone(),
                msg: to_binary(&TokenMsg::TransferFrom {
                    owner: env.message.sender.clone(),
                    recipient: env.contract.address.clone(),
                    amount: Uint128::from(amount),
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
            log("locked.sender", hex::encode(extend_address_to_32(&deps.api.canonical_address(&env.message.sender)?))),
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
    use std::time::{UNIX_EPOCH, SystemTime};
    use super::*;
    use crate::state::GuardianSetInfo;
    use cosmwasm_std::testing::{mock_dependencies, mock_env};
    use cosmwasm_std::HumanAddr;

    const ADDR_1: &str = "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe";
    const ADDR_2: &str = "8575Df9b3c97B4E267Deb92d93137844A97A0132";
    const VAA_VALID_TRANSFER: &str = "010000000001001063f503dd308134e0f158537f54c5799719f4fa2687dd276c72ef60ae0c82c47d4fb560545afaabdf60c15918e221763fd1892c75f2098c0ffd5db4af254a4501000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000101010101010101010101010101010101010101000000000000000000000000010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000";
    const VAA_VALID_GUARDIAN_SET_CHANGE: &str = "01000000000100d90d6f9cbc0458599cbe4d267bc9221b54955b94cb5cb338aeb845bdc9dd275f558871ea479de9cc0b44cfb2a07344431a3adbd2f98aa86f4e12ff4aba061b7f00000007d00100000001018575df9b3c97b4e267deb92d93137844a97a0132";

    const CANONICAL_LENGTH: usize = 20;

    fn do_init_default_guardians<S: Storage, A: Api, Q: Querier>(deps: &mut Extern<S, A, Q>) {
        do_init(deps, &vec![GuardianAddress::from(ADDR_1)]);
    }

    fn do_init<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        guardians: &Vec<GuardianAddress>,
    ) {
        let init_msg = InitMsg {
            initial_guardian_set: GuardianSetInfo {
                addresses: guardians.clone(),
                expiration_time: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + 1000,
            },
            guardian_set_expirity: 50,
            wrapped_asset_code_id: 999,
        };
        let env = mock_env(&HumanAddr::from("creator"), &[]);
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
        let env = mock_env(&HumanAddr::from("creator"), &[]);

        handle(deps, env, msg)
    }

    #[test]
    fn can_init() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_default_guardians(&mut deps);
    }

    #[test]
    fn valid_vaa_token_transfer() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_default_guardians(&mut deps);

        let messages = submit_vaa(&mut deps, VAA_VALID_TRANSFER).unwrap().messages;
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
    fn same_vaa_twice_error() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_default_guardians(&mut deps);

        let _ = submit_vaa(&mut deps, VAA_VALID_TRANSFER).unwrap();
        let e = submit_vaa(&mut deps, VAA_VALID_TRANSFER).unwrap_err();
        assert_eq!(e, ContractError::VaaAlreadyExecuted.std());
    }

    #[test]
    fn valid_vaa_guardian_set_change() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        do_init_default_guardians(&mut deps);

        let messages = submit_vaa(&mut deps, VAA_VALID_GUARDIAN_SET_CHANGE)
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
}
