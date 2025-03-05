use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Attribute, Binary};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

// TODO: figure out proper serde enum representation so we don't have to copy the core bridge execute message types
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SubmitVAA {
        vaa: Binary,
    },
    PostMessage {
        message: Binary,
        nonce: u32,
    },
    /// Submit a signed VAA to update the on-chain state.  If processing any of the VAAs
    /// returns an error, the entire transaction is aborted and none of the VAAs are committed.
    SubmitUpdateChannelChain {
        /// VAA to submit.  The VAA should be encoded in the standard wormhole
        /// wire format.
        vaa: Binary,
    },
}

/// This is the message we send over the IBC channel
#[cw_serde]
pub enum WormholeIbcPacketMsg {
    Publish { msg: Vec<Attribute> },
}

#[cfg(test)]
mod test {
    use cosmwasm_std::to_json_binary;
    use cw_wormhole::msg::ExecuteMsg as WormholeExecuteMsg;

    use super::ExecuteMsg;

    #[test]
    fn submit_vaa_serialization_matches() {
        let signed_vaa = "\
        080000000901007bfa71192f886ab6819fa4862e34b4d178962958d9b2e3d943\
        7338c9e5fde1443b809d2886eaa69e0f0158ea517675d96243c9209c3fe1d94d\
        5b19866654c6980000000b150000000500020001020304000000000000000000\
        000000000000000000000000000000000000000000000000000a0261626364";
        let signed_vaa = hex::decode(signed_vaa).unwrap();

        let wormhole_submit_vaa = WormholeExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone().into(),
        };
        let wormhole_msg = to_json_binary(&wormhole_submit_vaa).unwrap();

        let submit_vaa = ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.into(),
        };
        let msg = to_json_binary(&submit_vaa).unwrap();

        assert_eq!(wormhole_msg, msg);
    }
}
