use serde::{Deserialize, Serialize};

use crate::Chain;

/// Represents a governance action targeted at the wormchain ibc receiver contract.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Action {
    #[serde(rename = "1")]
    UpdateChainConnection {
        // an existing IBC connection ID
        #[serde(with = "crate::arraystring")]
        connection_id: bstr::BString,
        // the chain associated with this IBC connection_id
        chain_id: Chain,
    },
}

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
        Deserialize, Deserializer, Serialize, Serializer,
    };

    use crate::{
        ibc_receiver::{Action, GovernancePacket},
        Chain,
    };

    // MODULE = "IbcReceiver"
    const MODULE: [u8; 32] = [
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x49, 0x62, 0x63, 0x52, 0x65, 0x63, 0x65, 0x69, 0x76,
        0x65, 0x72,
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
                Err(Error::custom(
                    "invalid governance module, expected \"IbcReceiver\"",
                ))
            }
        }
    }

    // governance actions
    #[derive(Serialize, Deserialize)]
    struct UpdateChainConnection {
        #[serde(with = "crate::arraystring")]
        connection_id: bstr::BString,
        chain_id: Chain,
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
                    let UpdateChainConnection {
                        connection_id,
                        chain_id,
                    } = seq
                        .next_element()?
                        .ok_or_else(|| Error::invalid_length(3, &EXPECTING))?;

                    Action::UpdateChainConnection {
                        connection_id,
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
                                let UpdateChainConnection {
                                    connection_id,
                                    chain_id,
                                } = map.next_value()?;

                                Action::UpdateChainConnection {
                                    connection_id,
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
