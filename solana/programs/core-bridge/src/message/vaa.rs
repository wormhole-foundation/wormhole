use std::ops::{Deref, DerefMut};

use crate::types::{ChainId, ExternalAddress, Finality, Timestamp};
use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize, InitSpace};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct GuardianSignature {
    pub index: u8,
    pub rs: [u8; 64],
    pub recovery_id: u8,
}

impl WormDecode for GuardianSignature {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let index = u8::decode_reader(reader)?;
        let mut rs = [0; 64];
        reader.read_exact(&mut rs)?;
        let recovery_id = u8::decode_reader(reader)?;

        Ok(Self {
            index,
            rs,
            recovery_id,
        })
    }
}

impl WormEncode for GuardianSignature {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.index.encode(writer)?;
        writer.write_all(&self.rs)?;
        self.recovery_id.encode(writer)
    }
}

/// NOTE: `AnchorSerialize` and `AnchorDeserialize` are derived for this struct. But when this info
/// is read off the wire (i.e. from an encoded VAA), one must be careful about deserializing the VAA
/// using the `AnchorDeserialize` trait. The message info must be deserialized using `WormEncode`.
/// These Anchor traits are used to store the data in an account after the encoded VAA is decoded.
///
/// See `process_encoded_vaa` instruction handler for more info.
#[derive(Default, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct VaaV1MessageInfo {
    pub timestamp: Timestamp,
    pub nonce: u32,
    pub emitter_chain: ChainId,
    pub emitter_address: ExternalAddress,
    pub sequence: u64,
    pub finality: Finality,
}

impl WormDecode for VaaV1MessageInfo {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let timestamp = Timestamp::decode_reader(reader)?;
        let nonce = u32::decode_reader(reader)?;
        let emitter_chain = ChainId::decode_reader(reader)?;
        let emitter_address = ExternalAddress::decode_reader(reader)?;
        let sequence = u64::decode_reader(reader)?;
        let finality = Finality::decode_reader(reader)?;

        Ok(Self {
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            finality,
        })
    }
}

impl WormEncode for VaaV1MessageInfo {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.timestamp.encode(writer)?;
        self.nonce.encode(writer)?;
        self.emitter_chain.encode(writer)?;
        self.emitter_address.encode(writer)?;
        self.sequence.encode(writer)?;
        self.finality.encode(writer)
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct VaaV1MessageBody {
    pub info: VaaV1MessageInfo,
    pub payload: Vec<u8>,
}

impl Deref for VaaV1MessageBody {
    type Target = VaaV1MessageInfo;

    fn deref(&self) -> &Self::Target {
        &self.info
    }
}

impl DerefMut for VaaV1MessageBody {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.info
    }
}

// #[cfg(test)]
// mod test {
//     use anchor_lang::prelude::Result;

//     use super::*;

//     #[test]
//     fn encode_and_decode() -> Result<()> {
//         let encoded = vec![
//             0, 188, 97, 78, 0, 0, 164, 85, 0, 2, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173,
//             190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190,
//             239, 222, 173, 190, 239, 0, 0, 0, 0, 0, 0, 8, 0, 1, 65, 108, 108, 32, 121, 111, 117,
//             114, 32, 98, 97, 115, 101, 32, 97, 114, 101, 32, 98, 101, 108, 111, 110, 103, 32, 116,
//             111, 32, 117, 115, 46,
//         ];
//         let mut buf = encoded.as_slice();
//         let info = VaaV1MessageInfo::decode(&mut buf)?;

//         let mut payload = Vec::with_capacity(buf.len());
//         payload.extend_from_slice(buf);

//         let decoded = VaaV1MessageBody { info, payload };
//         assert_eq!(
//             decoded,
//             VaaV1MessageBody {
//                 info: VaaV1MessageInfo {
//                     timestamp: 12345678.into(),
//                     nonce: 42069,
//                     emitter_chain: 2.into(),
//                     emitter_address: [
//                         0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef,
//                         0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef,
//                         0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef
//                     ]
//                     .into(),
//                     sequence: 2048,
//                     finality: 1.into(),
//                 },
//                 payload: b"All your base are belong to us.".to_vec()
//             }
//         );

//         let mut new_encoded = Vec::with_capacity(encoded.len());
//         decoded.info.encode(&mut new_encoded)?;
//         new_encoded.extend_from_slice(&decoded.payload);
//         assert_eq!(encoded, new_encoded);

//         Ok(())
//     }
// }
