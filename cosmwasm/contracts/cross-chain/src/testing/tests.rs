use cosmwasm_std::{StdError, StdResult};

use cw_wormhole::{byte_utils::get_string_from_32, state::ParsedVAA};

use crate::state::{RegisterChainChannel, TransferPayload, UpgradeContract};

#[test]
fn verify_get_string_from_32_handles_null_strings() -> StdResult<()> {
    let data = hex::decode("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000").unwrap();
    let channel_id = get_string_from_32(&data);
    assert_eq!("", channel_id);
    Ok(())
}

#[test]
fn verify_get_string_from_32_handles_longer_strings() -> StdResult<()> {
    let long_string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789?!";
    let data = long_string.as_bytes();
    let channel_id = get_string_from_32(&data);
    assert_eq!(long_string, channel_id);
    Ok(())
}

#[test]
fn verify_register_chain_channel_deserialize() -> StdResult<()> {
    let data = hex::decode("00120000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d3138").unwrap();
    let RegisterChainChannel {
        chain_id,
        channel_id,
    } = RegisterChainChannel::deserialize(&data)?;
    assert_eq!(18, chain_id);
    assert_eq!("channel-18".to_string(), channel_id);
    Ok(())
}

#[test]
fn verify_upgrade_contract_deserialize() -> StdResult<()> {
    let data =
        hex::decode("0000000000000000000000000000000000000000000000000000000000001234").unwrap();
    let UpgradeContract { new_contract } = UpgradeContract::deserialize(&data)?;
    assert_eq!(0x1234, new_contract);
    Ok(())
}

/*
new Uint8Array(
  Buffer.from(
    JSON.stringify({
      basic_transfer: {
        chain_id: 18,
        recipient: "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"
      }
    })
  )
)

'{"basic_transfer":{"chain_id":18,"recipient":"terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"}}'
 */

#[test]
fn verify_transfer_payload_deserialize() -> StdResult<()> {
    let json = "{\"basic_transfer\":{\"chain_id\":18,\"recipient\":\"terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v\"}}";
    let payload: TransferPayload = serde_json_wasm::from_slice(json.as_bytes()).unwrap();
    match payload {
        TransferPayload::BasicTransfer {
            chain_id,
            recipient,
        } => {
            assert_eq!(18, chain_id);
            assert_eq!(
                "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v",
                recipient.to_string()
            );
            Ok(())
        }
        TransferPayload::BasicDeposit { amount: _ } => {
            Err(StdError::generic_err("wrong payload type"))
        }
    }
}

#[test]
fn verify_deposit_payload_deserialize() -> StdResult<()> {
    let json = "{\"basic_deposit\":{\"amount\":12345}}";
    let deposit: TransferPayload = serde_json_wasm::from_slice(json.as_bytes()).unwrap();
    match deposit {
        TransferPayload::BasicDeposit { amount } => {
            assert_eq!(12345, amount);
            Ok(())
        }
        TransferPayload::BasicTransfer {
            chain_id: _,
            recipient: _,
        } => Err(StdError::generic_err("wrong payload type")),
    }
}

#[test]
fn verify_parse_vaa() -> StdResult<()> {
    let vaa = "\
    0100000000010007e1a3ff6398cad8da78d1c7b402258c73a39682dd0523e84d846d1b36fb1de14b621366ea9402d62b901fc6\
    453cf4e4fd6f80f7869119003e50e8ad88ce0cab01000015a81e56010000020000000000000000000000000290fb167208af45\
    5bb137780163b7b7a9a10c16000000000000000001030000000000000000000000000000000000000000000000000000000005\
    f5e1000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002da7415da106b53b332ed33f38bbe\
    cf8d961d592525a851e67de290cbae3302900c2000000000000000000000000022d491bde2303f2f43325b2108d26f1eaba1e3\
    2b7b2262617369635f7472616e73666572223a7b22636861696e5f6964223a333130342c22726563697069656e74223a22746\
    57272613178343672716179346433637373713867787876717a387874366e776c7a34746432306b333876227d7d";

    let signed_vaa = hex::decode(vaa).unwrap();

    let parsed = ParsedVAA::deserialize(signed_vaa.as_slice())?;

    let version = 1u8;
    assert_eq!(parsed.version, version, "parsed.version != expected");

    let guardian_set_index = 0u32;
    assert_eq!(
        parsed.guardian_set_index, guardian_set_index,
        "parsed.guardian_set_index != expected"
    );

    let timestamp = 5544u32;
    assert_eq!(parsed.timestamp, timestamp, "parsed.timestamp != expected");

    let nonce = 508952832u32;
    assert_eq!(parsed.nonce, nonce, "parsed.nonce != expected");

    let len_signers = 1u8;
    assert_eq!(
        parsed.len_signers, len_signers,
        "parsed.len_signers != expected"
    );

    let emitter_chain = 2u16;
    assert_eq!(
        parsed.emitter_chain, emitter_chain,
        "parsed.emitter_chain != expected"
    );

    let emitter_address = "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16";
    let emitter_address = hex::decode(emitter_address).unwrap();
    assert_eq!(
        parsed.emitter_address, emitter_address,
        "parsed.emitter_address != expected"
    );

    let sequence = 0u64;
    assert_eq!(parsed.sequence, sequence, "parsed.sequence != expected");

    let consistency_level = 1u8;
    assert_eq!(
        parsed.consistency_level, consistency_level,
        "parsed.consistency_level != expected"
    );

    Ok(())
}
