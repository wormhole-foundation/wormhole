use cosmwasm_std::{
    Binary,
    StdResult,
};

use wormhole::state::ParsedVAA;

use crate::{
    contract::{
        build_asset_id,
        build_native_id,
    },
    state::{
        Action,
        TokenBridgeMessage,
        TransferInfo,
        TransferWithPayloadInfo,
    },
};

#[test]
fn binary_check() -> StdResult<()> {
    let x = vec![
        1u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 96u8, 180u8, 94u8, 195u8, 0u8, 0u8, 0u8,
        1u8, 0u8, 3u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 38u8, 229u8,
        4u8, 215u8, 149u8, 163u8, 42u8, 54u8, 156u8, 236u8, 173u8, 168u8, 72u8, 220u8, 100u8, 90u8,
        154u8, 159u8, 160u8, 215u8, 0u8, 91u8, 48u8, 44u8, 48u8, 44u8, 51u8, 44u8, 48u8, 44u8,
        48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
        44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 53u8, 55u8, 44u8, 52u8, 54u8, 44u8, 50u8, 53u8,
        53u8, 44u8, 53u8, 48u8, 44u8, 50u8, 52u8, 51u8, 44u8, 49u8, 48u8, 54u8, 44u8, 49u8, 50u8,
        50u8, 44u8, 49u8, 49u8, 48u8, 44u8, 49u8, 50u8, 53u8, 44u8, 56u8, 56u8, 44u8, 55u8, 51u8,
        44u8, 49u8, 56u8, 57u8, 44u8, 50u8, 48u8, 55u8, 44u8, 49u8, 48u8, 52u8, 44u8, 56u8, 51u8,
        44u8, 49u8, 49u8, 57u8, 44u8, 49u8, 50u8, 55u8, 44u8, 49u8, 57u8, 50u8, 44u8, 49u8, 52u8,
        55u8, 44u8, 56u8, 57u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8,
        48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
        44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8,
        48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
        44u8, 48u8, 44u8, 48u8, 44u8, 51u8, 44u8, 50u8, 51u8, 50u8, 44u8, 48u8, 44u8, 51u8, 44u8,
        48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
        44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 53u8, 51u8, 44u8, 49u8, 49u8, 54u8,
        44u8, 52u8, 56u8, 44u8, 49u8, 49u8, 54u8, 44u8, 49u8, 52u8, 57u8, 44u8, 49u8, 48u8, 56u8,
        44u8, 49u8, 49u8, 51u8, 44u8, 56u8, 44u8, 48u8, 44u8, 50u8, 51u8, 50u8, 44u8, 52u8, 57u8,
        44u8, 49u8, 53u8, 50u8, 44u8, 49u8, 44u8, 50u8, 56u8, 44u8, 50u8, 48u8, 51u8, 44u8, 50u8,
        49u8, 50u8, 44u8, 50u8, 50u8, 49u8, 44u8, 50u8, 52u8, 49u8, 44u8, 56u8, 53u8, 44u8, 49u8,
        48u8, 57u8, 93u8,
    ];
    let b = Binary::from(x.clone());
    let y: Vec<u8> = b.into();
    assert_eq!(x, y);
    Ok(())
}

#[test]
fn build_native_and_asset_ids() -> StdResult<()> {
    let denom = "uusd";
    let native_id = build_native_id(denom);

    let expected_native_id = vec![
        0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 117u8,
        117u8, 115u8, 100u8,
    ];
    assert_eq!(&native_id, &expected_native_id, "native_id != expected");

    // weth
    let chain = 2u16;
    let token_address = "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2";
    let token_address = hex::decode(token_address).unwrap();
    let asset_id = build_asset_id(chain, token_address.as_slice());

    let expected_asset_id = vec![
        171u8, 106u8, 233u8, 80u8, 14u8, 139u8, 124u8, 78u8, 181u8, 77u8, 142u8, 76u8, 109u8, 81u8,
        55u8, 100u8, 139u8, 159u8, 42u8, 85u8, 172u8, 234u8, 0u8, 114u8, 11u8, 82u8, 40u8, 40u8,
        50u8, 73u8, 211u8, 135u8,
    ];
    assert_eq!(&asset_id, &expected_asset_id, "asset_id != expected");
    Ok(())
}

#[test]
fn deserialize_transfer_vaa() -> StdResult<()> {
    let signed_vaa = "\
        010000000001003f3179d5bb17b6f2ecc13741ca3f78d922043e99e09975e390\
        4332d2418bb3f16d7ac93ca8401f8bed1cf9827bc806ecf7c5a283340f033bf4\
        72724abf1d274f00000000000000000000010000000000000000000000000000\
        00000000000000000000000000000000ffff0000000000000000000100000000\
        00000000000000000000000000000000000000000000000005f5e10001000000\
        0000000000000000000000000000000000000000000000007575736400030000\
        00000000000000000000f7f7dde848e7450a029cd0a9bd9bdae4b5147db30003\
        00000000000000000000000000000000000000000000000000000000000f4240";
    let signed_vaa = hex::decode(signed_vaa).unwrap();

    let parsed = ParsedVAA::deserialize(signed_vaa.as_slice())?;
    let message = TokenBridgeMessage::deserialize(&parsed.payload)?;
    assert_eq!(
        message.action,
        Action::TRANSFER,
        "message.action != expected"
    );

    let info = TransferInfo::deserialize(&message.payload)?;

    let amount = (0u128, 100_000_000u128);
    assert_eq!(info.amount, amount, "info.amount != expected");

    let token_address = "0100000000000000000000000000000000000000000000000000000075757364";
    let token_address = hex::decode(token_address).unwrap();
    assert_eq!(
        info.token_address, token_address,
        "info.token_address != expected"
    );

    let token_chain = 3u16;
    assert_eq!(
        info.token_chain, token_chain,
        "info.token_chain != expected"
    );

    let recipient = "000000000000000000000000f7f7dde848e7450a029cd0a9bd9bdae4b5147db3";
    let recipient = hex::decode(recipient).unwrap();
    assert_eq!(info.recipient, recipient, "info.recipient != expected");

    let recipient_chain = 3u16;
    assert_eq!(
        info.recipient_chain, recipient_chain,
        "info.recipient_chain != expected"
    );

    let fee = (0u128, 1_000_000u128);
    assert_eq!(info.fee, fee, "info.fee != expected");

    Ok(())
}

#[test]
fn deserialize_transfer_with_payload_vaa() -> StdResult<()> {
    let signed_vaa = "\
        010000000001002b0e392ebe370e718b91dcafbba21094efd8e7f1f12e28bd90\
        a178b4dfbbc708675152a3cd2edd20e8e018600026b73b6c6cbf02622903409e\
        8b48ab7fa30ef001000000010000000100010000000000000000000000000000\
        00000000000000000000000000000000ffff0000000000000002000300000000\
        00000000000000000000000000000000000000000000000005f5e10001000000\
        0000000000000000000000000000000000000000000000007575736400030000\
        000000000000000000008cec800d24df11e556e708461c98122df4a2c3b10003\
        00000000000000000000000000000000000000000000000000000000000f4240\
        416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
    let signed_vaa = hex::decode(signed_vaa).unwrap();

    let parsed = ParsedVAA::deserialize(signed_vaa.as_slice())?;
    let message = TokenBridgeMessage::deserialize(&parsed.payload)?;
    assert_eq!(
        message.action,
        Action::TRANSFER_WITH_PAYLOAD,
        "message.action != expected"
    );

    let info_with_payload = TransferWithPayloadInfo::deserialize(&message.payload)?;
    let info = info_with_payload.transfer_info;

    let amount = (0u128, 100_000_000u128);
    assert_eq!(info.amount, amount, "info.amount != expected");

    let token_address = "0100000000000000000000000000000000000000000000000000000075757364";
    let token_address = hex::decode(token_address).unwrap();
    assert_eq!(
        info.token_address, token_address,
        "info.token_address != expected"
    );

    let token_chain = 3u16;
    assert_eq!(
        info.token_chain, token_chain,
        "info.token_chain != expected"
    );

    let recipient = "0000000000000000000000008cec800d24df11e556e708461c98122df4a2c3b1";
    let recipient = hex::decode(recipient).unwrap();
    assert_eq!(info.recipient, recipient, "info.recipient != expected");

    let recipient_chain = 3u16;
    assert_eq!(
        info.recipient_chain, recipient_chain,
        "info.recipient_chain != expected"
    );

    let fee = (0u128, 1_000_000u128);
    assert_eq!(info.fee, fee, "info.fee != expected");

    let transfer_payload = "All your base are belong to us";
    let transfer_payload = transfer_payload.as_bytes();
    assert_eq!(
        info_with_payload.payload.as_slice(),
        transfer_payload,
        "info.payload != expected"
    );

    Ok(())
}
