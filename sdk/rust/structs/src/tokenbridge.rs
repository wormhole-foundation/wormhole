use bstr::BString;
use serde::{Deserialize, Serialize};

use crate::{Address, Amount, Chain};

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Message {
    #[serde(rename = "1")]
    Transfer {
        amount: Amount,
        token_address: Address,
        token_chain: Chain,
        recipient: Address,
        recipient_chain: Chain,
        fee: Amount,
    },
    #[serde(rename = "2")]
    AssetMeta {
        token_address: Address,
        token_chain: Chain,
        decimals: u8,
        #[serde(with = "crate::arraystring")]
        symbol: BString,
        #[serde(with = "crate::arraystring")]
        name: BString,
    },
    #[serde(rename = "3")]
    TransferWithPayload {
        amount: Amount,
        token_address: Address,
        token_chain: Chain,
        recipient: Address,
        recipient_chain: Chain,
        sender_address: Address,
        // The payload is directly appended to the message.
    },
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Action {
    ContractUpgrade {
        new_contract: Address,
    },
    RegisterChain {
        chain: Chain,
        emitter_address: Address,
    },
}

#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GovernancePacket {
    pub chain: Chain,
    pub action: Action,
}

// The wire format for GovernancePackets is wonky and doesn't lend itself well to auto-deriving
// Serialize / Deserialize so we implement it manually here.
mod governance_packet_impl {
    use std::fmt;

    use serde::{
        de::{Error, MapAccess, SeqAccess, Unexpected, Visitor},
        ser::SerializeStruct,
        Deserialize, Deserializer, Serialize, Serializer,
    };

    use crate::{
        tokenbridge::{Action, GovernancePacket},
        Address, Chain,
    };

    const MODULE: [u8; 32] = [
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x72, 0x69, 0x64,
        0x67, 0x65,
    ];

    struct Module;

    impl Serialize for Module {
        fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            MODULE.serialize(serializer)
        }
    }

    impl<'de> Deserialize<'de> for Module {
        fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            let arr = <[u8; 32]>::deserialize(deserializer)?;

            if arr == MODULE {
                Ok(Module)
            } else {
                let expected = format!("{MODULE:?}");
                Err(Error::invalid_value(Unexpected::Bytes(&arr), &&*expected))
            }
        }
    }

    #[derive(Serialize, Deserialize)]
    struct ContractUpgrade {
        new_contract: Address,
    }

    #[derive(Serialize, Deserialize)]
    struct RegisterChain {
        chain: Chain,
        emitter_address: Address,
    }

    impl Serialize for GovernancePacket {
        fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            let mut seq = serializer.serialize_struct("GovernancePacket", 4)?;
            seq.serialize_field("module", &Module)?;

            // The wire format encodes the action before the chain and then appends the actual
            // action payload.
            match self.action {
                Action::ContractUpgrade { new_contract } => {
                    seq.serialize_field("action", &1u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field("payload", &ContractUpgrade { new_contract })?;
                }
                Action::RegisterChain {
                    chain,
                    emitter_address,
                } => {
                    seq.serialize_field("action", &2u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field(
                        "payload",
                        &RegisterChain {
                            chain,
                            emitter_address,
                        },
                    )?;
                }
            }

            seq.end()
        }
    }

    struct GovernancePacketVisitor;

    impl<'de> Visitor<'de> for GovernancePacketVisitor {
        type Value = GovernancePacket;

        fn expecting(&self, f: &mut fmt::Formatter) -> fmt::Result {
            f.write_str("struct GovernancePacket")
        }

        #[inline]
        fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
        where
            A: SeqAccess<'de>,
        {
            static EXPECTING: &str = "struct GovernancePacket with 4 elements";

            let _: Module = seq
                .next_element()?
                .ok_or_else(|| Error::invalid_length(0, &EXPECTING))?;
            let act: u8 = seq
                .next_element()?
                .ok_or_else(|| Error::invalid_length(1, &EXPECTING))?;
            let chain = seq
                .next_element()?
                .ok_or_else(|| Error::invalid_length(2, &EXPECTING))?;

            let action = match act {
                1 => {
                    let ContractUpgrade { new_contract } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;

                    Action::ContractUpgrade { new_contract }
                }
                2 => {
                    let RegisterChain {
                        chain,
                        emitter_address,
                    } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;

                    Action::RegisterChain {
                        chain,
                        emitter_address,
                    }
                }
                v => {
                    return Err(Error::invalid_value(
                        Unexpected::Unsigned(v.into()),
                        &"one of 1, 2",
                    ))
                }
            };

            Ok(GovernancePacket { chain, action })
        }

        fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
        where
            A: MapAccess<'de>,
        {
            #[derive(Serialize, Deserialize)]
            #[serde(rename_all = "snake_case")]
            enum Field {
                Module,
                Action,
                Chain,
                Payload,
            }

            let mut module = None;
            let mut chain = None;
            let mut action = None;
            let mut payload = None;

            while let Some(key) = map.next_key::<Field>()? {
                match key {
                    Field::Module => {
                        if module.is_some() {
                            return Err(Error::duplicate_field("module"));
                        }

                        module = map.next_value::<Module>().map(Some)?;
                    }
                    Field::Action => {
                        if action.is_some() {
                            return Err(Error::duplicate_field("action"));
                        }

                        action = map.next_value::<u8>().map(Some)?;
                    }
                    Field::Chain => {
                        if chain.is_some() {
                            return Err(Error::duplicate_field("chain"));
                        }

                        chain = map.next_value().map(Some)?;
                    }
                    Field::Payload => {
                        if payload.is_some() {
                            return Err(Error::duplicate_field("payload"));
                        }

                        let a = action.as_ref().copied().ok_or_else(|| {
                            Error::custom("`action` must be known before deserializing `payload`")
                        })?;

                        let p = match a {
                            1 => {
                                let ContractUpgrade { new_contract } = map.next_value()?;

                                Action::ContractUpgrade { new_contract }
                            }
                            2 => {
                                let RegisterChain {
                                    chain,
                                    emitter_address,
                                } = map.next_value()?;

                                Action::RegisterChain {
                                    chain,
                                    emitter_address,
                                }
                            }
                            v => {
                                return Err(Error::custom(format_args!(
                                    "invalid action: {v}, expected one of: 1, 2"
                                )))
                            }
                        };

                        payload = Some(p);
                    }
                }
            }

            let chain = chain.ok_or_else(|| Error::missing_field("chain"))?;
            let action = payload.ok_or_else(|| Error::missing_field("payload"))?;

            Ok(GovernancePacket { chain, action })
        }
    }

    impl<'de> Deserialize<'de> for GovernancePacket {
        fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            const FIELDS: &'static [&'static str] = &["module", "action", "chain", "payload"];
            deserializer.deserialize_struct("GovernancePacket", FIELDS, GovernancePacketVisitor)
        }
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    pub fn transfer() {
        let payload = [
            0x01, 0xfa, 0x22, 0x8b, 0x3d, 0xf3, 0x59, 0x21, 0xa1, 0xc9, 0x39, 0xad, 0x9c, 0x54,
            0xe1, 0x2f, 0x87, 0xc6, 0x27, 0x25, 0xd1, 0xaf, 0x7c, 0x7b, 0x6a, 0xea, 0xbf, 0x48,
            0x14, 0xe7, 0x26, 0x56, 0xfa, 0xe8, 0xf2, 0xd2, 0x53, 0xd1, 0x08, 0x52, 0xa2, 0x6f,
            0x55, 0xae, 0x93, 0xb5, 0x9b, 0x46, 0x91, 0x3d, 0xdf, 0x89, 0x1c, 0xb5, 0x38, 0x75,
            0x5a, 0x45, 0x47, 0xde, 0x51, 0x12, 0x01, 0x5d, 0xc3, 0x00, 0x15, 0x7a, 0x1f, 0x37,
            0xe7, 0x14, 0x28, 0xbc, 0x04, 0x75, 0xd9, 0x26, 0x8e, 0x4b, 0x53, 0x06, 0xd3, 0xa1,
            0x22, 0x40, 0x13, 0xc4, 0x36, 0xbd, 0x1c, 0x23, 0xd7, 0x2a, 0x1c, 0x16, 0x46, 0x43,
            0x50, 0x00, 0x07, 0x48, 0xc7, 0x7c, 0x80, 0xbd, 0x11, 0xee, 0x41, 0x20, 0xb3, 0x52,
            0x3b, 0xd0, 0x0a, 0x2f, 0xdd, 0x02, 0xe8, 0xef, 0xfc, 0x7a, 0xfe, 0x3d, 0xd6, 0x73,
            0x2c, 0x5d, 0xa0, 0x6b, 0x08, 0x98, 0x6c,
        ];
        let msg = Message::Transfer {
            amount: Amount([
                0xfa, 0x22, 0x8b, 0x3d, 0xf3, 0x59, 0x21, 0xa1, 0xc9, 0x39, 0xad, 0x9c, 0x54, 0xe1,
                0x2f, 0x87, 0xc6, 0x27, 0x25, 0xd1, 0xaf, 0x7c, 0x7b, 0x6a, 0xea, 0xbf, 0x48, 0x14,
                0xe7, 0x26, 0x56, 0xfa,
            ]),
            token_address: Address([
                0xe8, 0xf2, 0xd2, 0x53, 0xd1, 0x08, 0x52, 0xa2, 0x6f, 0x55, 0xae, 0x93, 0xb5, 0x9b,
                0x46, 0x91, 0x3d, 0xdf, 0x89, 0x1c, 0xb5, 0x38, 0x75, 0x5a, 0x45, 0x47, 0xde, 0x51,
                0x12, 0x01, 0x5d, 0xc3,
            ]),
            token_chain: Chain::Sui,
            recipient: Address([
                0x7a, 0x1f, 0x37, 0xe7, 0x14, 0x28, 0xbc, 0x04, 0x75, 0xd9, 0x26, 0x8e, 0x4b, 0x53,
                0x06, 0xd3, 0xa1, 0x22, 0x40, 0x13, 0xc4, 0x36, 0xbd, 0x1c, 0x23, 0xd7, 0x2a, 0x1c,
                0x16, 0x46, 0x43, 0x50,
            ]),
            recipient_chain: Chain::Oasis,
            fee: Amount([
                0x48, 0xc7, 0x7c, 0x80, 0xbd, 0x11, 0xee, 0x41, 0x20, 0xb3, 0x52, 0x3b, 0xd0, 0x0a,
                0x2f, 0xdd, 0x02, 0xe8, 0xef, 0xfc, 0x7a, 0xfe, 0x3d, 0xd6, 0x73, 0x2c, 0x5d, 0xa0,
                0x6b, 0x08, 0x98, 0x6c,
            ]),
        };

        assert_eq!(payload.as_ref(), &vaa_payload::to_vec(&msg).unwrap());
        assert_eq!(msg, vaa_payload::from_slice(&payload).unwrap());
    }

    #[test]
    pub fn asset_meta() {
        let payload = [
            0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0xbe, 0xef, 0xfa, 0xce, 0x00, 0x02, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x42, 0x45, 0x45, 0x46, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x42, 0x65, 0x65, 0x66, 0x20, 0x66, 0x61, 0x63, 0x65, 0x20, 0x54, 0x6f, 0x6b,
            0x65, 0x6e,
        ];

        let msg = Message::AssetMeta {
            token_address: Address([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0xbe, 0xef, 0xfa, 0xce,
            ]),
            token_chain: Chain::Ethereum,
            decimals: 12,
            symbol: "BEEF".into(),
            name: "Beef face Token".into(),
        };

        assert_eq!(payload.as_ref(), &vaa_payload::to_vec(&msg).unwrap());
        assert_eq!(msg, vaa_payload::from_slice(&payload).unwrap());
    }

    #[test]
    pub fn transfer_with_payload() {
        let payload = [
            0x03, 0x99, 0x30, 0x29, 0x01, 0x67, 0xf3, 0x48, 0x6a, 0xf6, 0x4c, 0xdd, 0x07, 0x64,
            0xa1, 0x6a, 0xca, 0xcc, 0xc5, 0x64, 0x2a, 0x66, 0x71, 0xaa, 0x87, 0x98, 0x64, 0x0c,
            0x25, 0x1f, 0x39, 0x73, 0x25, 0x12, 0x98, 0x15, 0xe5, 0x79, 0xdb, 0xd8, 0x25, 0xfd,
            0x72, 0x58, 0x99, 0x97, 0xd1, 0x78, 0xb6, 0x2a, 0x57, 0x47, 0xab, 0xcf, 0x51, 0x62,
            0xa5, 0xb7, 0xec, 0x19, 0x4e, 0x03, 0x51, 0x41, 0x18, 0x00, 0x08, 0xa4, 0x7e, 0xa2,
            0x49, 0xd4, 0xf0, 0x56, 0x6f, 0xbd, 0x12, 0x77, 0xfe, 0xa8, 0x6e, 0x92, 0x81, 0x3a,
            0x35, 0x90, 0x3b, 0x29, 0x73, 0x4d, 0x75, 0x56, 0x0a, 0xfe, 0x70, 0xb0, 0xb9, 0x10,
            0x33, 0x00, 0x14, 0x47, 0x3e, 0x51, 0x74, 0x32, 0xbc, 0x3f, 0xb3, 0xf9, 0x82, 0xbb,
            0xf3, 0xc9, 0x43, 0x41, 0xcf, 0x74, 0x24, 0xff, 0xa4, 0x02, 0x35, 0xbe, 0xb1, 0x7c,
            0x47, 0x16, 0xba, 0xbc, 0xaa, 0xbe, 0x99, 0x54, 0xa2, 0x97, 0xbe, 0x2a, 0xed, 0xae,
            0x91, 0x95, 0xb9, 0x6a, 0x8a, 0xba, 0xbd, 0x3a, 0xc1, 0xa4, 0xd1, 0x96, 0x0a, 0xf9,
            0xab, 0x92, 0x42, 0x0d, 0x41, 0x33, 0xe5, 0xa8, 0x37, 0x32, 0xd3, 0xc3, 0xb8, 0xf3,
            0x93, 0xd7, 0xc0, 0x9e, 0xe8, 0x87, 0xae, 0x16, 0xbf, 0xfa, 0x5e, 0x70, 0xea, 0x36,
            0xa2, 0x82, 0x37, 0x1d, 0x46, 0x81, 0x94, 0x10, 0x34, 0xb1, 0xad, 0x0f, 0x4b, 0xc9,
            0x17, 0x1e, 0x91, 0x25, 0x11,
        ];
        let msg = Message::TransferWithPayload {
            amount: Amount([
                0x99, 0x30, 0x29, 0x01, 0x67, 0xf3, 0x48, 0x6a, 0xf6, 0x4c, 0xdd, 0x07, 0x64, 0xa1,
                0x6a, 0xca, 0xcc, 0xc5, 0x64, 0x2a, 0x66, 0x71, 0xaa, 0x87, 0x98, 0x64, 0x0c, 0x25,
                0x1f, 0x39, 0x73, 0x25,
            ]),
            token_address: Address([
                0x12, 0x98, 0x15, 0xe5, 0x79, 0xdb, 0xd8, 0x25, 0xfd, 0x72, 0x58, 0x99, 0x97, 0xd1,
                0x78, 0xb6, 0x2a, 0x57, 0x47, 0xab, 0xcf, 0x51, 0x62, 0xa5, 0xb7, 0xec, 0x19, 0x4e,
                0x03, 0x51, 0x41, 0x18,
            ]),
            token_chain: Chain::Algorand,
            recipient: Address([
                0xa4, 0x7e, 0xa2, 0x49, 0xd4, 0xf0, 0x56, 0x6f, 0xbd, 0x12, 0x77, 0xfe, 0xa8, 0x6e,
                0x92, 0x81, 0x3a, 0x35, 0x90, 0x3b, 0x29, 0x73, 0x4d, 0x75, 0x56, 0x0a, 0xfe, 0x70,
                0xb0, 0xb9, 0x10, 0x33,
            ]),
            recipient_chain: Chain::Osmosis,
            sender_address: Address([
                0x47, 0x3e, 0x51, 0x74, 0x32, 0xbc, 0x3f, 0xb3, 0xf9, 0x82, 0xbb, 0xf3, 0xc9, 0x43,
                0x41, 0xcf, 0x74, 0x24, 0xff, 0xa4, 0x02, 0x35, 0xbe, 0xb1, 0x7c, 0x47, 0x16, 0xba,
                0xbc, 0xaa, 0xbe, 0x99,
            ]),
        };

        assert_eq!(&payload[..133], &vaa_payload::to_vec(&msg).unwrap());
        let (actual, data) = vaa_payload::from_slice_with_payload(&payload).unwrap();
        assert_eq!(msg, actual);
        assert_eq!(&payload[133..], data);
    }
}
