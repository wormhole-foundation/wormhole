use cosmwasm_std::{Storage, Api, Querier, Extern, Env, StdResult, InitResponse, HandleResponse, Binary};

use crate::msg::{InitMsg, HandleMsg, QueryMsg};
use crate::state::{guardian_set, GuardianSetInfo, config, ConfigInfo, guardian_set_read, GuardianAddress, config_read};
use crate::error::ContractError;

use core::convert::TryInto;
use sha3::{Digest, Keccak256};
use k256::ecdsa::recoverable::Signature as RecoverableSignature;
use k256::ecdsa::recoverable::Id as RecoverableId;
use k256::ecdsa::Signature;
use k256::ecdsa::VerifyKey;
use k256::EncodedPoint;

use std::convert::TryFrom;

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    _env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {

    // Save general wormhole info
    let state = ConfigInfo {
        guardian_set_index: 0,
        guardian_set_expirity: msg.guardian_set_expirity,
    };
    config(&mut deps.storage).save(&state)?;

    // Add initial guardian set to storage
    guardian_set(&mut deps.storage).save(&state.guardian_set_index.to_le_bytes(), &msg.initial_guardian_set)?;

    Ok(InitResponse::default())
}

pub fn handle<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: HandleMsg,
) -> StdResult<HandleResponse> {
    match msg {
        HandleMsg::SubmitVAA {vaa} => {

            let version = vaa[0];
            if version != 1 {
                return ContractError::InvalidVersion.std_err();
            }

            // Load 4 bytes starting from index 1
            let vaa_guardian_set_index: u32 = u32::from_le_bytes(vaa[1..5].try_into().unwrap());
            let len_signers = vaa[5] as usize;
            let offset: usize = 6 + 66 * len_signers as usize;

            // Hash the body
            let body = &vaa[offset..];
            let mut hasher = Keccak256::new();
            hasher.update(body);
            let hash = hasher.finalize();
            // TODO: Check if VAA with this hash was already accepted

            // Load and check guardian set
            let guardian_set = guardian_set_read(&deps.storage).load(&vaa_guardian_set_index.to_le_bytes());
            let guardian_set: GuardianSetInfo = guardian_set.or(ContractError::InvalidGuardianSetIndex.std_err())?;

            if guardian_set.expiration_time == 0 || guardian_set.expiration_time > env.block.time {
                return ContractError::GuardianSetExpired.std_err();
            }
            if len_signers < (guardian_set.addresses.len() / 4) * 3 + 1 {
                return ContractError::NoQuorum.std_err();
            }

            // Verify guardian signatures
            let mut last_index: i32 = -1;
            for i in 0..len_signers {
                let index = vaa[6 + i * 66] as i32;
                if index <= last_index {
                    return Err(ContractError::WrongGuardianIndexOrder.std());
                }
                last_index = index;

                let signature = Signature::try_from(&vaa[7 + i * 66 .. 71 + i * 66]).or(ContractError::CannotDecodeSignature.std_err())?;
                let id = RecoverableId::new(vaa[71 + i * 66]).or(ContractError::CannotDecodeSignature.std_err())?;
                let recoverable_signature = RecoverableSignature::new(&signature, id).or(ContractError::CannotDecodeSignature.std_err())?;
                
                let verify_key = recoverable_signature.recover_verify_key_from_digest_bytes(&hash).or(ContractError::CannotRecoverKey.std_err())?;
                if !keys_equal(&verify_key, &guardian_set.addresses[index as usize]) {
                    return ContractError::GuardianSignatureError.std_err();
                }
            }

            // Signatures valid, apply VAA
            let action = vaa[offset + 4];
            let payload = &vaa[offset + 5..];

            match action {
                0x01 => {
                    let state = config_read(&deps.storage).load()?;
                    if vaa_guardian_set_index != state.guardian_set_index {
                        return ContractError::NotCurrentGuardianSet.std_err();
                    }
                    vaa_update_guardian_set(payload)?;
                },
                0x10 => {
                    vaa_transfer(payload)?;
                },
                _ => {
                    return ContractError::InvalidVAAAction.std_err();
                }
            }

            let res = HandleResponse {
                messages: vec![],
                log: vec![],
                data: None,
            };
            Ok(res)
        }
    }
}

fn vaa_update_guardian_set(_payload: &[u8]) -> StdResult<()> {
    // TODO: Apply new guardian set
    Ok(())
}

fn vaa_transfer(_payload: &[u8]) -> StdResult<()> {
    // TODO: Do transfer
    Ok(())
}

pub fn query<S: Storage, A: Api, Q: Querier>(
    _deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
    }
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
    use cosmwasm_std::testing::{mock_dependencies, mock_env};
    use cosmwasm_std::{HumanAddr};
    use crate::state::GuardianSetInfo;
    
    const CANONICAL_LENGTH: usize = 20;

    fn do_init<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        guardians: &Vec<GuardianAddress>)
    {
        let init_msg = InitMsg {
            initial_guardian_set: GuardianSetInfo {
                addresses: guardians.clone(),
                expiration_time: 100
            },
            guardian_set_expirity: 50
        };
        let env = mock_env(&HumanAddr::from("creator"), &[]);
        let res = init(deps, env, init_msg).unwrap();
        assert_eq!(0, res.messages.len());

        // TODO: Query and check contract state and guardians storage
    }

    fn submit_vaa<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        vaa: &str)
    {
        let msg = HandleMsg::SubmitVAA {
            vaa: hex::decode(vaa).expect("Decoding failed")
        };
        let env = mock_env(&HumanAddr::from("creator"), &[]);
        let res = handle(deps, env, msg).unwrap();
        assert_eq!(0, res.messages.len());
    }

    #[test]
    fn can_init() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let guardians = vec![GuardianAddress::from("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")];
        do_init(&mut deps, &guardians);
    }

    #[test]
    fn valid_vaa_token_transfer() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let guardians = vec![GuardianAddress::from("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")];
        do_init(&mut deps, &guardians);

        submit_vaa(&mut deps, "01000000000100454e7de661cd4386b1ce598a505825f8ed66fbc6a608393bae6257fef7370da27a2068240a902470bed6c0b1fa23d38e5d5958e2a422d59a0217fbe155638ed600000007d010000000380102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000");
    }
}