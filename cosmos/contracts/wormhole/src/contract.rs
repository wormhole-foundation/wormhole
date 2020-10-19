use cosmwasm_std::{Storage, Api, Querier, Extern, Env, StdResult, InitResponse, HandleResponse, Binary};

use crate::msg::{InitMsg, HandleMsg, QueryMsg};
use crate::state::{guardian_set, GuardianSetInfo, config, ConfigInfo};
use crate::error::ContractError;

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
    guardian_set(&mut deps.storage).save(&state.guardian_set_index.to_le_bytes(), &GuardianSetInfo::from(&deps.api, &msg.initial_guardian_set)?)?;

    Ok(InitResponse::default())
}

pub fn handle<S: Storage, A: Api, Q: Querier>(
    _deps: &mut Extern<S, A, Q>,
    _env: Env,
    msg: HandleMsg,
) -> StdResult<HandleResponse> {
    match msg {
        HandleMsg::SubmitVAA {vaa} => {

            let version = vaa[0];
            if version != 1 {
                return Err(ContractError::InvalidVersion.std());
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

pub fn query<S: Storage, A: Api, Q: Querier>(
    _deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env};
    use cosmwasm_std::{HumanAddr};
    use crate::msg::GuardianSetMsg;

    const CANONICAL_LENGTH: usize = 20;

    fn do_init<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        guardians: &Vec<HumanAddr>)
    {
        let init_msg = InitMsg {
            initial_guardian_set: GuardianSetMsg {
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

    #[test]
    fn can_init() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let guardians = vec![HumanAddr::from("guardian1"), HumanAddr::from("guardian2")];
        do_init(&mut deps, &guardians);
    }
}