//! Parsers for Standardized Relayer VAAs.
//!
//! The Standardized Relayer uses VAAs for delivery and redelivery instructions as well as governance.
//! The delivery VAA contain the sender and may contain an optional payload.

// Example devnet registration governance VAAs
// 010000000001007349c3b9b892432e4d371b37085547eea089533667cd16c0237f4012bc4a011562731487b93c31e9a9b7ab37df1e39ecad3a0a249f019759ef4afe66500db5180165d760ad00000001000100000000000000000000000000000000000000000000000000000000000000040000000000000001010000000000000000000000000000000000576f726d686f6c6552656c61796572010000000200000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5
// 01000000000100e4e0dd18bf7a1867027a6f4f9d53c07cab84950de5150e05d48ac7ba84e18f90394d4e825faac2b76b0ce95e34b0e3f91da75d457fc1dfae4720b1fbedc2e6540165d7609e00000001000100000000000000000000000000000000000000000000000000000000000000040000000000000001010000000000000000000000000000000000576f726d686f6c6552656c61796572010000000400000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5

use serde::{Deserialize, Serialize};

use crate::{Address, Chain};

/// Represents a governance action targeted at the standardized relayer.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Action {
    /// Registers an emitter address for a particular chain on a different chain.  An emitter
    /// address must be registered for a chain and must match the emitter address in the VAA before
    /// the standardized relayer will accept VAAs from that chain.
    #[serde(rename = "1")]
    RegisterChain {
        chain: Chain,
        emitter_address: Address,
    },

    /// Upgrades the standardized relayer contract to a new address.
    #[serde(rename = "2")]
    ContractUpgrade { new_contract: Address },
}

/// Represents the payload for a governance VAA targeted at the standardized relayer.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GovernancePacket {
    /// The chain on which the governance action should be carried out.
    pub chain: Chain,

    /// The actual governance action to be carried out.
    pub action: Action,
}

// MODULE = "WormholeRelayer"
pub const MODULE: [u8; 32] =
    *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00WormholeRelayer";

// The wire format for GovernancePackets is wonky and doesn't lend itself well to auto-deriving
// Serialize / Deserialize so we implement it manually here.
mod governance_packet_impl {
    use std::fmt;

    use serde::{
        de::{Error, MapAccess, SeqAccess, Visitor},
        ser::SerializeStruct,
        Deserialize, Deserializer, Serialize, Serializer,
    };

    use crate::{
        relayer::{Action, GovernancePacket, MODULE},
        Address, Chain,
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
                    "invalid governance module, expected \"WormholeRelayer\"",
                ))
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
            match self.action.clone() {
                Action::RegisterChain {
                    chain,
                    emitter_address,
                } => {
                    seq.serialize_field("action", &1u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field(
                        "payload",
                        &RegisterChain {
                            chain,
                            emitter_address,
                        },
                    )?;
                }
                Action::ContractUpgrade { new_contract } => {
                    seq.serialize_field("action", &2u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field("payload", &ContractUpgrade { new_contract })?;
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
                2 => {
                    let ContractUpgrade { new_contract } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;

                    Action::ContractUpgrade { new_contract }
                }
                v => {
                    return Err(Error::custom(format_args!(
                        "invalid value: {v}, expected one of 1, 2"
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
                                let RegisterChain {
                                    chain,
                                    emitter_address,
                                } = map.next_value()?;

                                Action::RegisterChain {
                                    chain,
                                    emitter_address,
                                }
                            }
                            2 => {
                                let ContractUpgrade { new_contract } = map.next_value()?;

                                Action::ContractUpgrade { new_contract }
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
    use crate::{vaa::Signature, Vaa, GOVERNANCE_EMITTER};

    use super::*;

    #[test]
    fn register_chain() {
        let buf = [
            0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x73, 0x49, 0xc3, 0xb9, 0xb8, 0x92, 0x43,
            0x2e, 0x4d, 0x37, 0x1b, 0x37, 0x08, 0x55, 0x47, 0xee, 0xa0, 0x89, 0x53, 0x36, 0x67,
            0xcd, 0x16, 0xc0, 0x23, 0x7f, 0x40, 0x12, 0xbc, 0x4a, 0x01, 0x15, 0x62, 0x73, 0x14,
            0x87, 0xb9, 0x3c, 0x31, 0xe9, 0xa9, 0xb7, 0xab, 0x37, 0xdf, 0x1e, 0x39, 0xec, 0xad,
            0x3a, 0x0a, 0x24, 0x9f, 0x01, 0x97, 0x59, 0xef, 0x4a, 0xfe, 0x66, 0x50, 0x0d, 0xb5,
            0x18, 0x01, 0x65, 0xd7, 0x60, 0xad, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x57, 0x6f, 0x72, 0x6d, 0x68, 0x6f, 0x6c, 0x65, 0x52, 0x65, 0x6c, 0x61, 0x79, 0x65,
            0x72, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x53, 0x85, 0x5d, 0x4b, 0x64, 0xe9, 0xa3, 0xcf, 0x59, 0xa8,
            0x4b, 0xc7, 0x68, 0xad, 0xa7, 0x16, 0xb5, 0x53, 0x6b, 0xc5,
        ];

        let vaa = Vaa {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![Signature {
                index: 0,
                signature: [
                    0x73, 0x49, 0xc3, 0xb9, 0xb8, 0x92, 0x43, 0x2e, 0x4d, 0x37, 0x1b, 0x37, 0x08,
                    0x55, 0x47, 0xee, 0xa0, 0x89, 0x53, 0x36, 0x67, 0xcd, 0x16, 0xc0, 0x23, 0x7f,
                    0x40, 0x12, 0xbc, 0x4a, 0x01, 0x15, 0x62, 0x73, 0x14, 0x87, 0xb9, 0x3c, 0x31,
                    0xe9, 0xa9, 0xb7, 0xab, 0x37, 0xdf, 0x1e, 0x39, 0xec, 0xad, 0x3a, 0x0a, 0x24,
                    0x9f, 0x01, 0x97, 0x59, 0xef, 0x4a, 0xfe, 0x66, 0x50, 0x0d, 0xb5, 0x18, 0x01,
                ],
            }],
            timestamp: 1708613805,
            nonce: 1,
            emitter_chain: Chain::Solana,
            emitter_address: GOVERNANCE_EMITTER,
            sequence: 1,
            consistency_level: 1,
            payload: GovernancePacket {
                chain: Chain::Any,
                action: Action::RegisterChain {
                    chain: Chain::Ethereum,
                    emitter_address: Address([
                        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                        0x53, 0x85, 0x5d, 0x4b, 0x64, 0xe9, 0xa3, 0xcf, 0x59, 0xa8, 0x4b, 0xc7,
                        0x68, 0xad, 0xa7, 0x16, 0xb5, 0x53, 0x6b, 0xc5,
                    ]),
                },
            },
        };

        assert_eq!(buf.as_ref(), &serde_wormhole::to_vec(&vaa).unwrap());
        assert_eq!(vaa, serde_wormhole::from_slice(&buf).unwrap());

        let encoded = serde_json::to_string(&vaa).unwrap();
        assert_eq!(vaa, serde_json::from_str(&encoded).unwrap());
    }
}
