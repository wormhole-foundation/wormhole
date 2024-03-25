//! Parsers for NTT Accountant Actions.
//!
//! NTT Accountant is a security mechanism for the NTT locking hubs.
//! It needs a modify_balance message to be able to correct for unforeseen events.

use bstr::BString;
use serde::{Deserialize, Serialize};

use crate::{accountant_modification::ModificationKind, Address, Amount, Chain};

/// Represents a governance action targeted at the Accountant.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Action {
    // Modify balance for accountant
    #[serde(rename = "1")]
    ModifyBalance {
        sequence: u64,
        chain_id: u16,
        token_chain: u16,
        token_address: Address,
        kind: ModificationKind,
        amount: Amount,
        #[serde(with = "crate::arraystring")]
        reason: BString,
    },
}

/// Represents the payload for a governance VAA targeted at the Accountant.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GovernancePacket {
    /// The chain on which the governance action should be carried out.
    pub chain: Chain,

    /// The actual governance action to be carried out.
    pub action: Action,
}

// MODULE = "NTTGlobalAccountant"
pub const MODULE: [u8; 32] =
    *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00NTTGlobalAccountant";

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
        ntt_accountant::{Action, GovernancePacket, MODULE},
        Address, Amount,
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
                    "invalid governance module, expected \"NTTGlobalAccountant\"",
                ))
            }
        }
    }

    #[derive(Serialize, Deserialize)]
    struct ModifyBalance {
        sequence: u64,
        chain_id: u16,
        token_chain: u16,
        token_address: Address,
        kind: super::ModificationKind,
        amount: Amount,
        #[serde(with = "crate::arraystring")]
        reason: bstr::BString,
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
                Action::ModifyBalance {
                    sequence,
                    chain_id,
                    token_chain,
                    token_address,
                    kind,
                    amount,
                    reason,
                } => {
                    seq.serialize_field("action", &1u8)?;
                    seq.serialize_field("chain", &self.chain)?;
                    seq.serialize_field(
                        "payload",
                        &ModifyBalance {
                            sequence,
                            chain_id,
                            token_chain,
                            token_address,
                            kind,
                            amount,
                            reason,
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
                    let ModifyBalance {
                        sequence,
                        chain_id,
                        token_chain,
                        token_address,
                        kind,
                        amount,
                        reason,
                    } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;
                    Action::ModifyBalance {
                        sequence,
                        chain_id,
                        token_chain,
                        token_address,
                        kind,
                        amount,
                        reason,
                    }
                }
                v => {
                    return Err(Error::custom(format_args!(
                        "invalid value {v}, expected one of 1"
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
                                let ModifyBalance {
                                    sequence,
                                    chain_id,
                                    token_chain,
                                    token_address,
                                    kind,
                                    amount,
                                    reason,
                                } = map.next_value()?;
                                Action::ModifyBalance {
                                    sequence,
                                    chain_id,
                                    token_chain,
                                    token_address,
                                    kind,
                                    amount,
                                    reason,
                                }
                            }
                            v => {
                                return Err(Error::custom(format_args!(
                                    "invalid action: {v}, expected one of: 1"
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
    fn modify_balance() {
        let buf = [
            0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xb0, 0x72, 0x50, 0x5b, 0x5b, 0x99, 0x9c,
            0x1d, 0x08, 0x90, 0x5c, 0x02, 0xe2, 0xb6, 0xb2, 0x83, 0x2e, 0xf7, 0x2c, 0x0b, 0xa6,
            0xc8, 0xdb, 0x4f, 0x77, 0xfe, 0x45, 0x7e, 0xf2, 0xb3, 0xd0, 0x53, 0x41, 0x0b, 0x1e,
            0x92, 0xa9, 0x19, 0x4d, 0x92, 0x10, 0xdf, 0x24, 0xd9, 0x87, 0xac, 0x83, 0xd7, 0xb6,
            0xf0, 0xc2, 0x1c, 0xe9, 0x0f, 0x8b, 0xc1, 0x86, 0x9d, 0xe0, 0x89, 0x8b, 0xda, 0x7e,
            0x98, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x1b, 0xfa, 0x00, 0x00, 0x00, 0x00,
            //  module = "NTTGlobalAccountant"
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x4E, 0x54, 0x54, 0x47,
            0x6C, 0x6F, 0x62, 0x61, 0x6C, 0x41, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x61, 0x6E,
            0x74, // action
            0x01, // chain
            0x00, 0x01, // sequence
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // chain_id
            0x00, 0x02, // token chain
            0x00, 0x03, // token address
            0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32,
            0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32, 0x32,
            0x32, 0x32, 0x32, 0x32, // kind
            0x01, // amount
            0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31,
            0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31,
            0x31, 0x31, 0x31, 0x31, // reason
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x41,
            0x42, 0x42, 0x43, 0x43,
        ];

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
                chain: Chain::Solana,
                action: Action::ModifyBalance {
                    sequence: 1,
                    chain_id: 2,
                    token_chain: 3,
                    token_address: Address([0x32u8; 32]),
                    kind: ModificationKind::Add,
                    amount: Amount([0x31u8; 32]),
                    reason: "AABBCC".into(),
                },
            },
        };

        assert_eq!(buf.as_ref(), &serde_wormhole::to_vec(&vaa).unwrap());
        assert_eq!(vaa, serde_wormhole::from_slice(&buf).unwrap());

        let encoded = serde_json::to_string(&vaa).unwrap();
        assert_eq!(vaa, serde_json::from_str(&encoded).unwrap());
    }
}
