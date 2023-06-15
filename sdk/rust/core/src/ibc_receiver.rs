use serde::{Deserialize, Serialize};

use crate::Chain;

/// Represents a governance action targeted at the wormchain ibc receiver contract.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Action {
    #[serde(rename = "1")]
    UpdateChannelChain {
        // an existing IBC channel ID
        #[serde(with = "crate::serde_array")]
        channel_id: [u8; 64],
        // the chain associated with this IBC channel_id
        chain_id: Chain,
    },
}

// MODULE = "IbcReceiver"
pub const MODULE: [u8; 32] = *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00IbcReceiver";

/// Represents the payload for a governance VAA targeted at the wormchain ibc receiver contract.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GovernancePacket {
    /// Describes the chain on which the governance action should be carried out.
    pub chain: Chain,

    /// The actual governance action to be carried out.
    pub action: Action,
}

mod governance_packet_impl {
    use std::fmt;

    use serde::{
        de::{Error, MapAccess, SeqAccess, Visitor},
        ser::SerializeStruct,
        Deserialize, Deserializer, Serialize, Serializer,
    };

    use crate::{
        ibc_receiver::{Action, GovernancePacket, MODULE},
        Chain,
    };

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
                Err(Error::custom(
                    "invalid governance module, expected \"IbcReceiver\"",
                ))
            }
        }
    }

    // governance actions
    #[derive(Serialize, Deserialize)]
    struct UpdateChannelChain {
        #[serde(with = "crate::serde_array")]
        channel_id: [u8; 64],
        chain_id: Chain,
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
            match self.action.clone() {
                Action::UpdateChannelChain {
                    channel_id,
                    chain_id,
                } => {
                    seq.serialize_field("action", &1u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field(
                        "payload",
                        &UpdateChannelChain {
                            channel_id,
                            chain_id,
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
                    let UpdateChannelChain {
                        channel_id,
                        chain_id,
                    } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;

                    Action::UpdateChannelChain {
                        channel_id,
                        chain_id,
                    }
                }
                v => {
                    return Err(Error::custom(format_args!(
                        "invalid value: {v}, expected 1"
                    )))
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
                                let UpdateChannelChain {
                                    channel_id,
                                    chain_id,
                                } = map.next_value()?;

                                Action::UpdateChannelChain {
                                    channel_id,
                                    chain_id,
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

            let _ = module.ok_or_else(|| Error::missing_field("module"))?;
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
            const FIELDS: &[&str] = &["module", "action", "chain", "payload"];
            deserializer.deserialize_struct("GovernancePacket", FIELDS, GovernancePacketVisitor)
        }
    }
}

#[cfg(test)]
mod test {
    use crate::{vaa::Signature, Chain, Vaa, GOVERNANCE_EMITTER};

    use super::{Action, GovernancePacket};

    #[test]
    fn happy_path() {
        let buf = [
            // version
            0x01, // guardian set index
            0x00, // signatures
            0x00, 0x00, 0x00, 0x01, 0x00, 0xb0, 0x72, 0x50, 0x5b, 0x5b, 0x99, 0x9c, 0x1d, 0x08,
            0x90, 0x5c, 0x02, 0xe2, 0xb6, 0xb2, 0x83, 0x2e, 0xf7, 0x2c, 0x0b, 0xa6, 0xc8, 0xdb,
            0x4f, 0x77, 0xfe, 0x45, 0x7e, 0xf2, 0xb3, 0xd0, 0x53, 0x41, 0x0b, 0x1e, 0x92, 0xa9,
            0x19, 0x4d, 0x92, 0x10, 0xdf, 0x24, 0xd9, 0x87, 0xac, 0x83, 0xd7, 0xb6, 0xf0, 0xc2,
            0x1c, 0xe9, 0x0f, 0x8b, 0xc1, 0x86, 0x9d, 0xe0, 0x89, 0x8b, 0xda, 0x7e, 0x98, 0x01,
            // timestamp
            0x00, 0x00, 0x00, 0x01, // nonce
            0x00, 0x00, 0x00, 0x01, // emitter chain
            0x00, 0x01, // emitter address
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x04, // sequence
            0x00, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x1b, 0xfa, // consistency
            0x00, //  module = "IbcReceiver"
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x49, 0x62, 0x63, 0x52, 0x65, 0x63, 0x65,
            0x69, 0x76, 0x65, 0x72, // action (IbcReceiverActionUpdateChannelChain)
            0x01, // target chain_id (unset)
            0x00, 0x00, // IBC channel_id for the mapping ("channel-0")
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x63,
            0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2d, 0x30, // IBC chain_id for the mapping
            0x00, 0x13,
        ];

        let channel_id_bytes: [u8; 64] =
            *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0";

        let vaa = Vaa {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![Signature {
                index: 0,
                signature: [
                    0xb0, 0x72, 0x50, 0x5b, 0x5b, 0x99, 0x9c, 0x1d, 0x08, 0x90, 0x5c, 0x02, 0xe2,
                    0xb6, 0xb2, 0x83, 0x2e, 0xf7, 0x2c, 0x0b, 0xa6, 0xc8, 0xdb, 0x4f, 0x77, 0xfe,
                    0x45, 0x7e, 0xf2, 0xb3, 0xd0, 0x53, 0x41, 0x0b, 0x1e, 0x92, 0xa9, 0x19, 0x4d,
                    0x92, 0x10, 0xdf, 0x24, 0xd9, 0x87, 0xac, 0x83, 0xd7, 0xb6, 0xf0, 0xc2, 0x1c,
                    0xe9, 0x0f, 0x8b, 0xc1, 0x86, 0x9d, 0xe0, 0x89, 0x8b, 0xda, 0x7e, 0x98, 0x01,
                ],
            }],
            timestamp: 1,
            nonce: 1,
            emitter_chain: Chain::Solana,
            emitter_address: GOVERNANCE_EMITTER,
            sequence: 20_716_538,
            consistency_level: 0,
            payload: GovernancePacket {
                chain: Chain::Any,
                action: Action::UpdateChannelChain {
                    channel_id: channel_id_bytes,
                    chain_id: Chain::Injective,
                },
            },
        };

        assert_eq!(buf.as_ref(), &serde_wormhole::to_vec(&vaa).unwrap());
        assert_eq!(vaa, serde_wormhole::from_slice(&buf).unwrap());

        let encoded = serde_json::to_string(&vaa).unwrap();
        assert_eq!(vaa, serde_json::from_str(&encoded).unwrap());
    }
}
