use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Response, StdError, StdResult,
};
use tokenbridge::msg::QueryMsg;

pub fn instantiate(_: DepsMut, _: Env, _: MessageInfo, _: Empty) -> StdResult<Response> {
    Ok(Response::new())
}

pub fn execute(_: DepsMut, _: Env, _: MessageInfo, _: Empty) -> StdResult<Response> {
    Err(StdError::GenericErr {
        msg: "execute not implemented".into(),
    })
}

pub fn query(_: Deps, _: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::ChainRegistration { chain } => to_binary(&vec![chain as u8; 32]),
        _ => Err(StdError::GenericErr {
            msg: "unimplemented query message".into(),
        }),
    }
}
