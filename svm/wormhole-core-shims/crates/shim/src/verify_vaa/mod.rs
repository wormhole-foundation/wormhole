mod post_signatures;

pub use post_signatures::*;

use solana_program::keccak::HASH_BYTES;
use wormhole_svm_definitions::{make_anchor_discriminator, GUARDIAN_SIGNATURE_LENGTH};

pub enum VerifyVaaShimInstruction<'ix, const CONTIGUOUS: bool> {
    PostSignatures(PostSignaturesData<'ix, CONTIGUOUS>),
    VerifyVaa([u8; HASH_BYTES]),
    CloseSignatures,
}

impl<'ix, const CONTIGUOUS: bool> VerifyVaaShimInstruction<'ix, CONTIGUOUS> {
    pub const CLOSE_SIGNATURES_SELECTOR: [u8; 8] =
        make_anchor_discriminator(b"global:close_signatures");
    pub const POST_SIGNATURES_SELECTOR: [u8; 8] =
        make_anchor_discriminator(b"global:post_signatures");
    pub const VERIFY_VAA_SELECTOR: [u8; 8] = make_anchor_discriminator(b"global:verify_vaa");
}

impl<'ix> VerifyVaaShimInstruction<'ix, false> {
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
                out.extend_from_slice(unsafe {
                    core::slice::from_raw_parts(
                        data.guardian_signatures.as_ptr() as *const u8,
                        encoded_signatures_len,
                    )
                });

                out
            }
            Self::VerifyVaa(hash) => {
                let mut out = Vec::with_capacity({
                    8 // selector
                    + HASH_BYTES
                });
                out.extend_from_slice(&Self::VERIFY_VAA_SELECTOR);
                out.extend_from_slice(hash);

                out
            }
            Self::CloseSignatures => Self::CLOSE_SIGNATURES_SELECTOR.to_vec(),
        }
    }
}

impl<'ix> VerifyVaaShimInstruction<'ix, true> {
    #[inline]
    pub fn deserialize(data: &'ix [u8]) -> Option<Self> {
        if data.len() < 8 {
            return None;
        }

        match data[..8].try_into().unwrap() {
            Self::POST_SIGNATURES_SELECTOR => {
                PostSignaturesData::deserialize(&data[8..]).map(Self::PostSignatures)
            }
            Self::VERIFY_VAA_SELECTOR => data[8..(8 + HASH_BYTES)]
                .try_into()
                .ok()
                .map(Self::VerifyVaa),
            Self::CLOSE_SIGNATURES_SELECTOR => Some(Self::CloseSignatures),
            _ => None,
        }
    }
}
