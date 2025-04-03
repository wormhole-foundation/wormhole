//! Wormhole Verify VAA Shim program instructions and related types.

mod close_signatures;
mod post_signatures;
mod verify_hash;

pub use close_signatures::*;
pub use post_signatures::*;
pub use verify_hash::*;

use wormhole_svm_definitions::{make_anchor_discriminator, Hash, GUARDIAN_SIGNATURE_LENGTH};

/// Instructions for the Verify VAA Shim program.
pub enum VerifyVaaShimInstruction<'ix, const CONTIGUOUS: bool> {
    PostSignatures(PostSignaturesData<'ix, CONTIGUOUS>),
    VerifyHash(VerifyHashData),
    CloseSignatures,
}

impl<const CONTIGUOUS: bool> VerifyVaaShimInstruction<'_, CONTIGUOUS> {
    pub const CLOSE_SIGNATURES_SELECTOR: [u8; 8] =
        make_anchor_discriminator(b"global:close_signatures");
    pub const POST_SIGNATURES_SELECTOR: [u8; 8] =
        make_anchor_discriminator(b"global:post_signatures");
    pub const VERIFY_HASH_SELECTOR: [u8; 8] = make_anchor_discriminator(b"global:verify_hash");
}

impl VerifyVaaShimInstruction<'_, false> {
    /// Serialize the instruction data.
    #[inline]
    pub fn to_vec(&self) -> Vec<u8> {
        match self {
            Self::PostSignatures(data) => {
                let guardian_signatures_len = data.guardian_signatures.len();
                let encoded_signatures_len = guardian_signatures_len * GUARDIAN_SIGNATURE_LENGTH;

                let mut out = Vec::with_capacity({
                    8 // selector
                    + PostSignaturesData::<false>::MINIMUM_SIZE
                    + encoded_signatures_len // guardian_signatures
                });
                out.extend_from_slice(&Self::POST_SIGNATURES_SELECTOR);
                out.extend_from_slice(&data.guardian_set_index.to_le_bytes());
                out.push(data.total_signatures);
                out.extend_from_slice(&(guardian_signatures_len as u32).to_le_bytes());
                for signature in data.guardian_signatures.iter() {
                    out.extend_from_slice(signature);
                }

                out
            }
            Self::VerifyHash(data) => {
                let mut out = Vec::with_capacity({
                    8 // selector
                    + VerifyHashData::SIZE
                });
                out.extend_from_slice(&Self::VERIFY_HASH_SELECTOR);
                out.push(data.guardian_set_bump);
                out.extend_from_slice(&data.digest.0);

                out
            }
            Self::CloseSignatures => Self::CLOSE_SIGNATURES_SELECTOR.to_vec(),
        }
    }
}

impl<'ix> VerifyVaaShimInstruction<'ix, true> {
    /// Deserialize the instruction data.
    #[inline]
    pub fn deserialize(data: &'ix [u8]) -> Option<Self> {
        if data.len() < 8 {
            return None;
        }

        match data[..8].try_into().unwrap() {
            Self::POST_SIGNATURES_SELECTOR => {
                PostSignaturesData::deserialize(&data[8..]).map(Self::PostSignatures)
            }
            Self::VERIFY_HASH_SELECTOR => {
                VerifyHashData::deserialize(&data[8..]).map(Self::VerifyHash)
            }
            Self::CLOSE_SIGNATURES_SELECTOR => Some(Self::CloseSignatures),
            _ => None,
        }
    }
}
