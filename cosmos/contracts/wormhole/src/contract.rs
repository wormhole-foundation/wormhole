use cosmwasm_std::{
    log, to_binary, Api, Binary, CosmosMsg, Env, Extern, HandleResponse, InitResponse, Querier,
    StdResult, Storage, Uint128, WasmMsg,
};

use crate::byte_utils::ByteUtils;
use crate::error::ContractError;
use crate::msg::{HandleMsg, InitMsg, QueryMsg};
use crate::state::{
    config, config_read, guardian_set, guardian_set_read, wrapped_asset, wrapped_asset_read,
    ConfigInfo, GuardianAddress, GuardianSetInfo,
};

use cw20_wrapped::msg::HandleMsg as WrappedMsg;
use cw20_wrapped::msg::InitMsg as WrappedInit;
use cw20_wrapped::msg::{InitHook, InitMint};

use k256::ecdsa::recoverable::Id as RecoverableId;
use k256::ecdsa::recoverable::Signature as RecoverableSignature;
use k256::ecdsa::Signature;
use k256::ecdsa::VerifyKey;
use k256::EncodedPoint;
use sha3::{Digest, Keccak256};

use std::convert::TryFrom;

// Chain ID of Cosmos
const CHAIN_ID: u8 = 0x80;

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    _env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {
    // Save general wormhole info
    let state = ConfigInfo {
        guardian_set_index: 0,
        guardian_set_expirity: msg.guardian_set_expirity,
        wrapped_asset_code_id: msg.wrapped_asset_code_id,
    };
    config(&mut deps.storage).save(&state)?;

    // Add initial guardian set to storage
    guardian_set(&mut deps.storage).save(
        &state.guardian_set_index.to_le_bytes(),
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
        HandleMsg::SubmitVAA { vaa } => handle_submit_vaa(deps, env, &vaa),
        HandleMsg::RegisterAssetHook { asset_id } => handle_register_asset(deps, env, &asset_id),
    }
}

/// Process VAA message signed by quardians
fn handle_submit_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &[u8],
) -> StdResult<HandleResponse> {
    let version = data.get_u8(0);
    if version != 1 {
        return ContractError::InvalidVersion.std_err();
    }

    // Load 4 bytes starting from index 1
    let vaa_guardian_set_index: u32 = data.get_u32(1);
    let len_signers = data.get_u8(5) as usize;
    let offset: usize = 6 + 66 * len_signers as usize;

    // Hash the body
    let body = &data[offset..];
    let mut hasher = Keccak256::new();
    hasher.update(body);
    let hash = hasher.finalize();
    // TODO: Check if VAA with this hash was already accepted

    // Load and check guardian set
    let guardian_set = guardian_set_read(&deps.storage).load(&vaa_guardian_set_index.to_le_bytes());
    let guardian_set: GuardianSetInfo =
        guardian_set.or(ContractError::InvalidGuardianSetIndex.std_err())?;

    if guardian_set.expiration_time == 0 || guardian_set.expiration_time > env.block.time {
        return ContractError::GuardianSetExpired.std_err();
    }
    if len_signers < (guardian_set.addresses.len() / 4) * 3 + 1 {
        return ContractError::NoQuorum.std_err();
    }

    // Verify guardian signatures
    let mut last_index: i32 = -1;
    for i in 0..len_signers {
        let index = data.get_u8(6 + i * 66) as i32;
        if index <= last_index {
            return Err(ContractError::WrongGuardianIndexOrder.std());
        }
        last_index = index;

        let signature = Signature::try_from(&data[7 + i * 66..71 + i * 66])
            .or(ContractError::CannotDecodeSignature.std_err())?;
        let id = RecoverableId::new(data.get_u8(71 + i * 66))
            .or(ContractError::CannotDecodeSignature.std_err())?;
        let recoverable_signature = RecoverableSignature::new(&signature, id)
            .or(ContractError::CannotDecodeSignature.std_err())?;

        let verify_key = recoverable_signature
            .recover_verify_key_from_digest_bytes(&hash)
            .or(ContractError::CannotRecoverKey.std_err())?;
        if !keys_equal(&verify_key, &guardian_set.addresses[index as usize]) {
            return ContractError::GuardianSignatureError.std_err();
        }
    }

    // Signatures valid, apply VAA
    let action = data.get_u8(offset + 4);
    let payload = &data[offset + 5..];

    match action {
        0x01 => {
            let state = config_read(&deps.storage).load()?;
            if vaa_guardian_set_index != state.guardian_set_index {
                return ContractError::NotCurrentGuardianSet.std_err();
            }
            vaa_update_guardian_set(payload)
        }
        0x10 => vaa_transfer(deps, env, payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
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

fn vaa_update_guardian_set(_payload: &[u8]) -> StdResult<HandleResponse> {
    // TODO: Apply new guardian set
    Ok(HandleResponse {
        messages: vec![],
        log: vec![],
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
                        asset_address: asset_address.to_vec(),
                        decimals: data.get_u8(103),
                        mint: Some(InitMint {
                            recipient: deps.api.human_address(&target_address)?,
                            amount: Uint128::from(amount),
                        }),
                        init_hook: Some(InitHook {
                            contract_addr: env.contract.address,
                            msg: to_binary(&HandleMsg::RegisterAssetHook {
                                asset_id: asset_id.to_vec(),
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
                msg: to_binary(&WrappedMsg::Transfer {
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

pub fn query<S: Storage, A: Api, Q: Querier>(
    _deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {}
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
    for (ai, bi) in a.iter().zip(b.iter()) {
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

    const CANONICAL_LENGTH: usize = 20;

    fn do_init<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        guardians: &Vec<GuardianAddress>,
    ) {
        let init_msg = InitMsg {
            initial_guardian_set: GuardianSetInfo {
                addresses: guardians.clone(),
                expiration_time: 100,
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
    ) -> Vec<CosmosMsg> {
        let msg = HandleMsg::SubmitVAA {
            vaa: hex::decode(vaa).expect("Decoding failed"),
        };
        let env = mock_env(&HumanAddr::from("creator"), &[]);
        let res = handle(deps, env, msg).unwrap();

        res.messages
    }

    #[test]
    fn can_init() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let guardians = vec![GuardianAddress::from(
            "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
        )];
        do_init(&mut deps, &guardians);
    }

    #[test]
    fn valid_vaa_token_transfer() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let guardians = vec![GuardianAddress::from(
            "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
        )];
        do_init(&mut deps, &guardians);

        let messages = submit_vaa(&mut deps, "010000000001005468beb21caff68710b2af2d60a986245bf85099509b6babe990a6c32456b44b3e2e9493e3056b7d5892957e14beab24be02dab77ed6c8915000e4a1267f78f400000007d01000000038018002010400000000000000000000000000000000000000000000000000000000000101010101010101010101010101010101010101000000000000000000000000010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000");
        assert_eq!(1, messages.len());
        let msg = &messages[0];
        match msg {
            CosmosMsg::Wasm(wasm_msg) => {
                match wasm_msg {
                    WasmMsg::Instantiate {
                        code_id, msg: _, send, label
                    } => {
                        assert_eq!(*code_id, 999);
                        assert_eq!(*label, None);
                        assert_eq!(*send, vec![]);
                    },
                    _ => panic!("Wrong message type")
                }
            },
            _ => panic!("Wrong message type")
        }
    }
}
