use cosmwasm_std::{
    has_coins,
    log,
    to_binary,
    Api,
    BankMsg,
    Binary,
    Coin,
    CosmosMsg,
    Env,
    Extern,
    HandleResponse,
    HumanAddr,
    InitResponse,
    Querier,
    StdError,
    StdResult,
    Storage,
};

use crate::{
    byte_utils::{
        extend_address_to_32,
        ByteUtils,
    },
    error::ContractError,
    msg::{
        GetAddressHexResponse,
        GetStateResponse,
        GuardianSetInfoResponse,
        HandleMsg,
        InitMsg,
        QueryMsg,
    },
    state::{
        config,
        config_read,
        guardian_set_get,
        guardian_set_set,
        sequence_read,
        sequence_set,
        vaa_archive_add,
        vaa_archive_check,
        ConfigInfo,
        GovernancePacket,
        GuardianAddress,
        GuardianSetInfo,
        GuardianSetUpgrade,
        ParsedVAA,
        SetFee,
        TransferFee,
    },
};

use k256::{
    ecdsa::{
        recoverable::{
            Id as RecoverableId,
            Signature as RecoverableSignature,
        },
        Signature,
        VerifyKey,
    },
    EncodedPoint,
};
use sha3::{
    Digest,
    Keccak256,
};

use generic_array::GenericArray;
use std::convert::TryFrom;

// Chain ID of Terra
const CHAIN_ID: u16 = 3;

// Lock assets fee amount and denomination
const FEE_AMOUNT: u128 = 10000;
pub const FEE_DENOMINATION: &str = "uluna";

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    _env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {
    // Save general wormhole info
    let state = ConfigInfo {
        gov_chain: msg.gov_chain,
        gov_address: msg.gov_address.as_slice().to_vec(),
        guardian_set_index: 0,
        guardian_set_expirity: msg.guardian_set_expirity,
        fee: Coin::new(FEE_AMOUNT, FEE_DENOMINATION), // 0.01 Luna (or 10000 uluna) fee by default
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
        HandleMsg::PostMessage { message, nonce } => {
            handle_post_message(deps, env, &message.as_slice(), nonce)
        }
        HandleMsg::SubmitVAA { vaa } => handle_submit_vaa(deps, env, vaa.as_slice()),
    }
}

/// Process VAA message signed by quardians
fn handle_submit_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &[u8],
) -> StdResult<HandleResponse> {
    let state = config_read(&deps.storage).load()?;

    let vaa = parse_and_verify_vaa(&deps.storage, data, env.block.time)?;
    vaa_archive_add(&mut deps.storage, vaa.hash.as_slice())?;

    if state.gov_chain == vaa.emitter_chain && state.gov_address == vaa.emitter_address {
        if state.guardian_set_index != vaa.guardian_set_index {
            return Err(StdError::generic_err(
                "governance VAAs must be signed by the current guardian set",
            ));
        }
        return handle_governance_payload(deps, env, &vaa.payload);
    }

    ContractError::InvalidVAAAction.std_err()
}

fn handle_governance_payload<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let gov_packet = GovernancePacket::deserialize(&data)?;

    let module = String::from_utf8(gov_packet.module).unwrap();
    let module: String = module.chars().filter(|c| c != &'\0').collect();

    if module != "Core" {
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != CHAIN_ID {
        return Err(StdError::generic_err(
            "the governance VAA is for another chain",
        ));
    }

    match gov_packet.action {
        // 1 is reserved for upgrade / migration
        2u8 => vaa_update_guardian_set(deps, env, &gov_packet.payload),
        3u8 => handle_set_fee(deps, env, &gov_packet.payload),
        4u8 => handle_transfer_fee(deps, env, &gov_packet.payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

/// Parses raw VAA data into a struct and verifies whether it contains sufficient signatures of an
/// active guardian set i.e. is valid according to Wormhole consensus rules
fn parse_and_verify_vaa<S: Storage>(
    storage: &S,
    data: &[u8],
    block_time: u64,
) -> StdResult<ParsedVAA> {
    let vaa = ParsedVAA::deserialize(data)?;

    if vaa.version != 1 {
        return ContractError::InvalidVersion.std_err();
    }

    // Check if VAA with this hash was already accepted
    if vaa_archive_check(storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }

    // Load and check guardian set
    let guardian_set = guardian_set_get(storage, vaa.guardian_set_index);
    let guardian_set: GuardianSetInfo =
        guardian_set.or_else(|_| ContractError::InvalidGuardianSetIndex.std_err())?;

    if guardian_set.expiration_time != 0 && guardian_set.expiration_time < block_time {
        return ContractError::GuardianSetExpired.std_err();
    }
    if (vaa.len_signers as usize) < guardian_set.quorum() {
        return ContractError::NoQuorum.std_err();
    }

    // Verify guardian signatures
    let mut last_index: i32 = -1;
    let mut pos = ParsedVAA::HEADER_LEN;

    for _ in 0..vaa.len_signers {
        if pos + ParsedVAA::SIGNATURE_LEN > data.len() {
            return ContractError::InvalidVAA.std_err();
        }
        let index = data.get_u8(pos) as i32;
        if index <= last_index {
            return ContractError::WrongGuardianIndexOrder.std_err();
        }
        last_index = index;

        let signature = Signature::try_from(
            &data[pos + ParsedVAA::SIG_DATA_POS
                ..pos + ParsedVAA::SIG_DATA_POS + ParsedVAA::SIG_DATA_LEN],
        )
        .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;
        let id = RecoverableId::new(data.get_u8(pos + ParsedVAA::SIG_RECOVERY_POS))
            .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;
        let recoverable_signature = RecoverableSignature::new(&signature, id)
            .or_else(|_| ContractError::CannotDecodeSignature.std_err())?;

        let verify_key = recoverable_signature
            .recover_verify_key_from_digest_bytes(GenericArray::from_slice(vaa.hash.as_slice()))
            .or_else(|_| ContractError::CannotRecoverKey.std_err())?;

        let index = index as usize;
        if index >= guardian_set.addresses.len() {
            return ContractError::TooManySignatures.std_err();
        }
        if !keys_equal(&verify_key, &guardian_set.addresses[index]) {
            return ContractError::GuardianSignatureError.std_err();
        }
        pos += ParsedVAA::SIGNATURE_LEN;
    }

    Ok(vaa)
}

fn vaa_update_guardian_set<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    /* Payload format
    0   uint32 new_index
    4   uint8 len(keys)
    5   [][20]uint8 guardian addresses
    */

    let mut state = config_read(&deps.storage).load()?;

    let GuardianSetUpgrade {
        new_guardian_set_index,
        new_guardian_set,
    } = GuardianSetUpgrade::deserialize(&data)?;

    if new_guardian_set_index != state.guardian_set_index + 1 {
        return ContractError::GuardianSetIndexIncreaseError.std_err();
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

pub fn handle_set_fee<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let set_fee_msg = SetFee::deserialize(&data)?;

    // Save new fees
    let mut state = config_read(&mut deps.storage).load()?;
    state.fee = set_fee_msg.fee;
    config(&mut deps.storage).save(&state)?;

    Ok(HandleResponse {
        messages: vec![],
        log: vec![
            log("action", "fee_change"),
            log("new_fee.amount", state.fee.amount),
            log("new_fee.denom", state.fee.denom),
        ],
        data: None,
    })
}

pub fn handle_transfer_fee<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let transfer_msg = TransferFee::deserialize(&data)?;

    Ok(HandleResponse {
        messages: vec![CosmosMsg::Bank(BankMsg::Send {
            from_address: env.contract.address,
            to_address: deps.api.human_address(&transfer_msg.recipient)?,
            amount: vec![transfer_msg.amount],
        })],
        log: vec![],
        data: None,
    })
}

fn handle_post_message<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    message: &[u8],
    nonce: u32,
) -> StdResult<HandleResponse> {
    let state = config_read(&deps.storage).load()?;
    let fee = state.fee;

    // Check fee
    if !has_coins(env.message.sent_funds.as_ref(), &fee) {
        return ContractError::FeeTooLow.std_err();
    }

    let emitter = extend_address_to_32(&deps.api.canonical_address(&env.message.sender)?);

    let sequence = sequence_read(&deps.storage, emitter.as_slice());
    sequence_set(&mut deps.storage, emitter.as_slice(), sequence + 1)?;

    Ok(HandleResponse {
        messages: vec![],
        log: vec![
            log("message.message", hex::encode(message)),
            log("message.sender", hex::encode(emitter)),
            log("message.chain_id", CHAIN_ID),
            log("message.nonce", nonce),
            log("message.sequence", sequence),
            log("message.block_time", env.block.time),
        ],
        data: None,
    })
}

pub fn query<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
        QueryMsg::GuardianSetInfo {} => to_binary(&query_guardian_set_info(deps)?),
        QueryMsg::VerifyVAA { vaa, block_time } => to_binary(&query_parse_and_verify_vaa(
            deps,
            &vaa.as_slice(),
            block_time,
        )?),
        QueryMsg::GetState {} => to_binary(&query_state(deps)?),
        QueryMsg::QueryAddressHex { address } => to_binary(&query_address_hex(deps, &address)?),
    }
}

pub fn query_guardian_set_info<S: Storage, A: Api, Q: Querier>(
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

pub fn query_parse_and_verify_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    data: &[u8],
    block_time: u64,
) -> StdResult<ParsedVAA> {
    parse_and_verify_vaa(&deps.storage, data, block_time)
}

// returns the hex of the 32 byte address we use for some address on this chain
pub fn query_address_hex<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    address: &HumanAddr,
) -> StdResult<GetAddressHexResponse> {
    Ok(GetAddressHexResponse {
        hex: hex::encode(extend_address_to_32(&deps.api.canonical_address(&address)?)),
    })
}

pub fn query_state<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
) -> StdResult<GetStateResponse> {
    let state = config_read(&deps.storage).load()?;
    let res = GetStateResponse { fee: state.fee };
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
