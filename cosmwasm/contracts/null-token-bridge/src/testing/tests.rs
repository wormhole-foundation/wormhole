use std::convert::TryInto;

use cosmwasm_std::{
    Binary,
    StdResult,
};

use wormhole::state::ParsedVAA;

use crate::{
    state::{
        Action,
        TokenBridgeMessage,
        TransferInfo,
        TransferWithPayloadInfo,
    },
    token_address::ExternalTokenId,
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
    let external_id_uluna = ExternalTokenId::from_bank_token(&"uluna".to_string())?;

    let expected_external_id: [u8; 32] = [1, 250, 108, 111, 188, 54, 216, 194, 69, 176, 168, 82, 164, 62, 181, 214, 68, 232, 180, 196, 119, 178, 123, 250, 185, 83, 124, 16, 148, 89, 57, 218];
    assert_eq!(
        &external_id_uluna.serialize(),
        &expected_external_id,
        "external_id != expected"
    );

    // weth
    let token_address = "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2";
    let token_address: [u8; 32] = hex::decode(token_address)
        .unwrap()
        .as_slice()
        .try_into()
        .unwrap();
    let external_id_weth = ExternalTokenId::from_foreign_token(token_address);

    let expected_asset_id: [u8; 32] = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 192, 42, 170, 57, 178, 35, 254, 141, 10, 14, 92, 79, 39, 234, 217, 8, 60, 117, 108, 194];
    assert_eq!(
        &external_id_weth.serialize(),
        &expected_asset_id,
        "asset_id != expected"
    );
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
        info.token_address.serialize().to_vec(),
        token_address,
        "info.token_address != expected"
    );

    let token_chain = 3u16;
    assert_eq!(
        info.token_chain, token_chain,
        "info.token_chain != expected"
    );

    let recipient = "000000000000000000000000f7f7dde848e7450a029cd0a9bd9bdae4b5147db3";
    let recipient = hex::decode(recipient).unwrap();
    assert_eq!(
        info.recipient.to_vec(),
        recipient,
        "info.recipient != expected"
    );

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

// ┌──────────────────────────────────────────────────────────────────────────────┐
// │ Wormhole VAA v1         │ nonce: 2080370133       │ time: 0                  │
// │ guardian set #0         │ #4568529024235897313    │ consistency: 32          │
// ├──────────────────────────────────────────────────────────────────────────────┤
// │ Signature:                                                                   │
// │   #0: 2565e7ae10421624fd81118855acda893e752aeeef31c13fbfc417591ada...        │
// ├──────────────────────────────────────────────────────────────────────────────┤
// │ Emitter: 11111111111111111111111111111115 (Solana)                           │
// ╞══════════════════════════════════════════════════════════════════════════════╡
// │ Token transfer with payload (aka payload 3)                                  │
// │ Amount: 1.0                                                                  │
// │ Token: terra1qqqqqqqqqqqqqqqqqqqqqqqqqp6h2umyswfh6y (Terra)                  │
// │ Recipient: terra13nkgqrfymug724h8pprpexqj9h629sa3ncw7sh (Terra)              │
// │ From: 1399a4e782b935d2bb36b97586d3df8747b07dc66902d807eed0ae99e00ed256       │
// ╞══════════════════════════════════════════════════════════════════════════════╡
// │ Custom payload:                                                              │
// │ Length: 30 (0x1e) bytes                                                      │
// │ 0000:   41 6c 6c 20  79 6f 75 72  20 62 61 73  65 20 61 72   All your base ar│
// │ 0010:   65 20 62 65  6c 6f 6e 67  20 74 6f 20  75 73         e belong to us  │
// └──────────────────────────────────────────────────────────────────────────────┘

    let signed_vaa = "\
        010000000001002565e7ae10421624fd81118855acda893e752aeeef31c13fbf\
        c417591ada039822195a1321a72cc4bac1c6031e0595f1c1361ca2a30d941a41\
        95fad8020d43d500000000007bffedd500010000000000000000000000000000\
        0000000000000000000000000000000000043f66acf143a481e1200300000000\
        00000000000000000000000000000000000000000000000005f5e10000000000\
        0000000000000000000000000000000000000000000000007575736400030000\
        000000000000000000008cec800d24df11e556e708461c98122df4a2c3b10003\
        1399a4e782b935d2bb36b97586d3df8747b07dc66902d807eed0ae99e00ed256\
        416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
    let signed_vaa = hex::decode(signed_vaa).unwrap();

    let parsed = ParsedVAA::deserialize(signed_vaa.as_slice())?;
    let message = TokenBridgeMessage::deserialize(&parsed.payload)?;
    assert_eq!(
        message.action,
        Action::TRANSFER_WITH_PAYLOAD,
        "message.action != expected"
    );

    let info = TransferWithPayloadInfo::deserialize(&message.payload)?;

    let amount = (0u128, 100_000_000u128);
    assert_eq!(info.amount, amount, "info.amount != expected");

    let token_address = "0000000000000000000000000000000000000000000000000000000075757364";
    let token_address = hex::decode(token_address).unwrap();
    assert_eq!(
        info.token_address.serialize().to_vec(),
        token_address,
        "info.token_address != expected"
    );

    let token_chain = 3u16;
    assert_eq!(
        info.token_chain, token_chain,
        "info.token_chain != expected"
    );

    let recipient = "0000000000000000000000008cec800d24df11e556e708461c98122df4a2c3b1";
    let recipient = hex::decode(recipient).unwrap();
    assert_eq!(
        info.recipient.to_vec(),
        recipient,
        "info.recipient != expected"
    );

    let sender = "1399a4e782b935d2bb36b97586d3df8747b07dc66902d807eed0ae99e00ed256";
    let sender = hex::decode(sender).unwrap();
    assert_eq!(
        info.sender_address.to_vec(),
        sender,
        "info.sender != expected"
    );

    let recipient_chain = 3u16;
    assert_eq!(
        info.recipient_chain, recipient_chain,
        "info.recipient_chain != expected"
    );


    let transfer_payload = "All your base are belong to us";
    let transfer_payload = transfer_payload.as_bytes();
    assert_eq!(
        info.payload.as_slice(),
        transfer_payload,
        "info.payload != expected"
    );

    Ok(())
}
