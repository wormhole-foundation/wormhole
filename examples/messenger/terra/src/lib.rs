use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use cosmwasm_std::{
    entry_point,
    DepsMut,
    Env,
    MessageInfo,
    Response,
    StdError,
    StdResult,
};
use wormhole_sdk::{
    parse_vaa,
    post_message,
};

use messenger_common::Message;

mod messages;
use messages::*;


#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    Ok(Response::default().add_attribute("version", msg.version))
}


#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        // Emit a new message targetting an address on a foreign chain. The message is emitted via
        // Wormhole and routed by the Guardians to the destination contract.
        ExecuteMsg::SendMessage { nonce, nick, text } => Ok(Response::default()
            .add_attribute("action", "send_message")
            .add_message(post_message(
                nonce,
                &Message { nick, text }
                    .try_to_vec()
                    .map_err(|_| StdError::generic_err("Encoding Failed"))?,
            )?)),

        // Receive a VAA containing a message from another chain. The message is stored in the
        // Terra contract state and can be read out via QueryMsg.
        ExecuteMsg::RecvMessage { vaa } => {
            // Parse VAA and decode Payload into message.
            let vaa = parse_vaa(deps, env, &vaa)?;
            let msg = Message::try_from_slice(&vaa.payload)
                .map_err(|_| StdError::generic_err("Invalid Message"))?;

            Ok(Response::default()
                .add_attribute("action", "receive_message")
                .add_attribute("nick", msg.nick)
                .add_attribute("text", msg.text))
        }
    }
}

#[cfg(test)]
mod testing {
    use cosmwasm_std::testing::{
        mock_dependencies,
        mock_env,
        mock_info,
    };
    use cosmwasm_std::{
        Attribute,
        Binary,
        CosmosMsg,
        SubMsg,
        WasmMsg,
    };
    use messenger_common::Message;

    use super::{
        execute,
        instantiate,
        ExecuteMsg,
        InstantiateMsg,
    };

    #[test]
    fn test_send_message() {
        // Test Messages
        let instantiate_msg = InstantiateMsg {
            version: "1.0.0".to_string(),
        };

        let send_msg = ExecuteMsg::SendMessage {
            nonce: 0,
            nick:  "Bob".to_string(),
            text:  "Hello Alice".to_string(),
        };

        // Instantiate Contract
        let mut deps = mock_dependencies(&[]);
        let env = mock_env();
        let info = mock_info("addr0000", &[]);
        instantiate(deps.as_mut(), env, info, instantiate_msg).unwrap();

        // Send a Message
        let info = mock_info("addr0000", &[]);
        let result = execute(deps.as_mut(), mock_env(), info, send_msg).unwrap();

        // Contract should have emitted a Msg targetting the wormhole contract.
        assert_eq!(
            result.messages,
            vec![SubMsg::new(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: wormhole_sdk::id().to_string(),
                funds:         vec![],
                msg:           Binary::from(&[
                    123, 34, 112, 111, 115, 116, 95, 109, 101, 115, 115, 97, 103, 101, 34, 58, 123,
                    34, 109, 101, 115, 115, 97, 103, 101, 34, 58, 34, 87, 122, 77, 115, 77, 67,
                    119, 119, 76, 68, 65, 115, 78, 106, 89, 115, 77, 84, 69, 120, 76, 68, 107, 52,
                    76, 68, 69, 120, 76, 68, 65, 115, 77, 67, 119, 119, 76, 68, 99, 121, 76, 68,
                    69, 119, 77, 83, 119, 120, 77, 68, 103, 115, 77, 84, 65, 52, 76, 68, 69, 120,
                    77, 83, 119, 122, 77, 105, 119, 50, 78, 83, 119, 120, 77, 68, 103, 115, 77, 84,
                    65, 49, 76, 68, 107, 53, 76, 68, 69, 119, 77, 86, 48, 61, 34, 44, 34, 110, 111,
                    110, 99, 101, 34, 58, 48, 125, 125
                ]),
            }))]
        );
    }

    #[test]
    fn test_recv_message() {
        // Test Messages
        let instantiate_msg = InstantiateMsg {
            version: "1.0.0".to_string(),
        };

        // Submit a pre-encoded Message VAA.
        let recv_msg = ExecuteMsg::RecvMessage {
            vaa: Binary::from(&[
                0x01, // Version
                0x00, 0x00, 0x00, 0x00, // Guardian Set Index
                0x01, // Guardian Signature Len
                0x00, // Guardian 0.
                0xb0, // Recovery 0.
                // Signature 0
                0x72, 0x50, 0x5b, 0x5b, 0x99, 0x9c, 0x1d, 0x08, 0x90, 0x5c, 0x02, 0xe2, 0xb6, 0xb2,
                0x83, 0x2e, 0xf7, 0x2c, 0x0b, 0xa6, 0xc8, 0xdb, 0x4f, 0x77, 0xfe, 0x45, 0x7e, 0xf2,
                0xb3, 0xd0, 0x53, 0x41, 0x0b, 0x1e, 0x92, 0xa9, 0x19, 0x4d, 0x92, 0x10, 0xdf, 0x24,
                0xd9, 0x87, 0xac, 0x83, 0xd7, 0xb6, 0xf0, 0xc2, 0x1c, 0xe9, 0x0f, 0x8b, 0xc1, 0x86,
                0x9d, 0xe0, 0x89, 0x8b, 0xda, 0x7e, 0x98, 0x01, 0x00, 0x00, 0x00,
                0x01, // Timestamp
                0x00, 0x00, 0x00, 0x01, // Nonce
                0x00, 0x01, // Chain Solana
                // Emitter Solana
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x1b,
                0xfa, // Sequence
                0x00, // Consistency
                // Payload
                5, 0, 0, 0, 65, 108, 105, 99, 101, 9, 0, 0, 0, 72, 101, 108, 108, 111, 32, 66, 111,
                98,
            ]),
        };

        use borsh::BorshSerialize;
        println!(
            "{:?}",
            Message {
                nick: "Alice".to_string(),
                text: "Hello Bob".to_string(),
            }
            .try_to_vec()
        );

        // Instantiate Contract
        let mut deps = mock_dependencies(&[]);
        let env = mock_env();
        let info = mock_info("addr0000", &[]);
        instantiate(deps.as_mut(), env, info, instantiate_msg).unwrap();

        // Receive a Message
        let info = mock_info("addr0000", &[]);
        let result = execute(deps.as_mut(), mock_env(), info, recv_msg).unwrap();
        assert_eq!(
            result.attributes,
            vec![
                Attribute {
                    key:   "action".to_string(),
                    value: "receive_message".to_string(),
                },
                Attribute {
                    key:   "nick".to_string(),
                    value: "Alice".to_string(),
                },
                Attribute {
                    key:   "text".to_string(),
                    value: "Hello Bob".to_string(),
                }
            ]
        );
    }
}
