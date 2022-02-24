pub mod msg;
pub mod state;

use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

pub use cosmwasm_std::to_binary;
use cosmwasm_std::Empty;

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema, Debug, Default)]
pub struct Trait {
    pub display_type: Option<String>,
    pub trait_type: String,
    pub value: String,
}

pub type Extension = Option<Empty>;

pub type Cw721MetadataContract<'a> = cw721_base::Cw721Contract<'a, Extension, Empty>;
pub type ExecuteMsg = cw721_base::ExecuteMsg<Extension>;

#[cfg(not(feature = "library"))]
pub mod entry {

    use std::convert::TryInto;

    use crate::msg::{
        InstantiateMsg,
        WrappedAssetInfoResponse,
    };
    pub use crate::{
        msg::QueryMsg,
        state::{
            wrapped_asset_info,
            wrapped_asset_info_read,
            WrappedAssetInfo,
        },
    };

    use super::*;

    use cosmwasm_std::{
        entry_point,
        to_binary,
        Binary,
        CosmosMsg,
        Deps,
        DepsMut,
        Env,
        MessageInfo,
        Response,
        StdError,
        StdResult,
        WasmMsg,
    };
    use cw721::Cw721Query;

    // version info for migration info
    const CONTRACT_NAME: &str = "crates.io:cw721-wrapped";
    const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

    // This is a simple type to let us handle empty extensions

    // This makes a conscious choice on the various generics used by the contract
    #[entry_point]
    pub fn instantiate(
        deps: DepsMut,
        env: Env,
        info: MessageInfo,
        msg: InstantiateMsg,
    ) -> StdResult<Response> {
        let base = Cw721MetadataContract::default();

        cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

        let contract_info = cw721::ContractInfoResponse {
            name: msg.name,
            symbol: msg.symbol,
        };
        base.contract_info.save(deps.storage, &contract_info)?;
        let minter = deps.api.addr_validate(&msg.minter)?;
        base.minter.save(deps.storage, &minter)?;

        // save wrapped asset info
        let data =
            WrappedAssetInfo {
                asset_chain: msg.asset_chain,
                asset_address: msg.asset_address.to_vec().try_into().map_err(
                    |_err| -> StdError {
                        StdError::GenericErr {
                            msg: "WrongSize".to_string(),
                        }
                    },
                )?,
                bridge: deps.api.addr_canonicalize(&info.sender.as_str())?,
            };
        wrapped_asset_info(deps.storage).save(&data)?;

        if let Some(mint_msg) = msg.mint {
            execute(deps, env, info, ExecuteMsg::Mint(mint_msg))
                .map_err(|e| StdError::generic_err(format!("{}", e)))?;
        }

        if let Some(hook) = msg.init_hook {
            Ok(
                Response::new().add_message(CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr: hook.contract_addr,
                    msg: hook.msg,
                    funds: vec![],
                })),
            )
        } else {
            Ok(Response::default())
        }
    }

    #[entry_point]
    pub fn execute(
        deps: DepsMut,
        env: Env,
        info: MessageInfo,
        msg: ExecuteMsg,
    ) -> Result<Response, cw721_base::ContractError> {
        Cw721MetadataContract::default().execute(deps, env, info, msg)
    }

    #[entry_point]
    pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
        let base = Cw721MetadataContract::default();
        match msg {
            QueryMsg::WrappedAssetInfo {} => to_binary(&query_wrapped_asset_info(deps)?),
            QueryMsg::OwnerOf {
                token_id,
                include_expired,
            } => {
                to_binary(&base.owner_of(deps, env, token_id, include_expired.unwrap_or(false))?)
            }
            QueryMsg::Approval {
                token_id,
                spender,
                include_expired,
            } => to_binary(&base.approval(
                deps,
                env,
                token_id,
                spender,
                include_expired.unwrap_or(false),
            )?),
            QueryMsg::Approvals {
                token_id,
                include_expired,
            } => {
                to_binary(&base.approvals(deps, env, token_id, include_expired.unwrap_or(false))?)
            }
            QueryMsg::AllOperators {
                owner,
                include_expired,
                start_after,
                limit,
            } => to_binary(&base.operators(
                deps,
                env,
                owner,
                include_expired.unwrap_or(false),
                start_after,
                limit,
            )?),
            QueryMsg::NumTokens {} => to_binary(&base.num_tokens(deps)?),
            QueryMsg::Tokens {
                owner,
                start_after,
                limit,
            } => to_binary(&base.tokens(deps, owner, start_after, limit)?),
            QueryMsg::AllTokens { start_after, limit } => {
                to_binary(&base.all_tokens(deps, start_after, limit)?)
            }
            QueryMsg::Minter {} => to_binary(&base.minter(deps)?),
            QueryMsg::ContractInfo {} => to_binary(&base.contract_info(deps)?),
            QueryMsg::NftInfo { token_id } => to_binary(&base.nft_info(deps, token_id)?),
            QueryMsg::AllNftInfo {
                token_id,
                include_expired,
            } => to_binary(&base.all_nft_info(
                deps,
                env,
                token_id,
                include_expired.unwrap_or(false),
            )?),
        }
    }

    pub fn query_wrapped_asset_info(deps: Deps) -> StdResult<WrappedAssetInfoResponse> {
        let info = wrapped_asset_info_read(deps.storage).load()?;
        Ok(WrappedAssetInfoResponse {
            asset_chain: info.asset_chain,
            asset_address: info.asset_address,
            bridge: deps.api.addr_humanize(&info.bridge)?,
        })
    }
}
