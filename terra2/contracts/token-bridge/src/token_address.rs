/// Represent the external view of a token address.
/// This is the value that goes into the wormhole VAA.
///
/// TODO: implement and document the new behaviour using hashing and storage.
/// Currently this is just the old implementation that looks at the first byte
/// and if it's 1 it assumes it's a native token (which is incorrect, because
/// now the contract address space is 32 bytes instead of 20, so the first byte
/// could legitimately be 1)
pub struct ExternalTokenId {
    address: [u8; 32],
}

impl ExternalTokenId {
    // TODO: update this (see above)
    pub fn to_token_id(&self) -> TokenId {
        let marker_byte = self.address[0];
        match marker_byte {
            1 => {
                let mut token_address = self.address.clone();
                token_address[0] = 0;
                let mut denom = token_address.to_vec();
                denom.retain(|&c| c != 0);
                let denom = String::from_utf8(denom).unwrap();
                TokenId::Native { denom }
            }
            _ => TokenId::CW20 {
                address: self.address,
            },
        }
    }
}

impl From<&[u8; 32]> for ExternalTokenId {
    fn from(address: &[u8; 32]) -> Self {
        Self { address: *address }
    }
}

/// Internal view of a token id.
pub enum TokenId {
    Native { denom: String },
    CW20 { address: [u8; 32] },
}
