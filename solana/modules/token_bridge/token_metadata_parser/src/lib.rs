// when I say 0 dependencies, I mean 0 dependencies
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub struct Pubkey(pub [u8; 32]);

impl Pubkey {
    pub fn new(bytes: [u8; 32]) -> Self {
        Self(bytes)
    }
    pub fn from_slice(s: &[u8]) -> Option<Self> {
        if s.len() == 32 {
            let mut a = [0u8; 32];
            a.copy_from_slice(s);
            Some(Self(a))
        } else {
            None
        }
    }
    pub fn is_zero(&self) -> bool {
        self.0.iter().all(|&b| b == 0)
    }
}

#[derive(Debug, PartialEq)]
pub enum MintMetadata {
    None,
    External(MetadataPointer),
    Embedded(TokenMetadata),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MetadataPointer {
    pub authority: Option<Pubkey>,
    /// Where metadata lives. If equals the mint pubkey, metadata is embedded in the mint (TokenMetadata extension).
    pub metadata_address: Pubkey,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TokenMetadata {
    pub update_authority: Option<Pubkey>,
    /// Mint this metadata belongs to (usually the mint itself)
    pub mint: Pubkey,
    pub name: String,
    pub symbol: String,
    pub uri: String,
    /// Arbitrary additional (key, value) pairs.
    pub additional_metadata: Vec<(String, String)>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ParseError {
    NotAMintAccount,    // data < 82
    NotMintAccountType, // account-type byte != Mint(1)
    NoEmbeddedMetadata, // pointer points to self but no embedded metadata found
    UnexpectedEnd,
    UnexpectedLength,
    LengthMismatch,
    InvalidUtf8,
}

const BASE_MINT_LEN: usize = 82;
const BASE_MINT_LEN_V2: usize = 165;
const ACCOUNT_TYPE_OFFSET: usize = BASE_MINT_LEN_V2;
const ACCOUNT_TYPE_MINT: u8 = 1;

// TLV header sizes
const TLV_TYPE_SIZE: usize = 2; // u16 LE
const TLV_LEN_SIZE: usize = 2; // u16 LE

// Extension type discriminants we care about:
const EXT_METADATA_POINTER: u16 = 18;
const EXT_TOKEN_METADATA: u16 = 19;
const EXT_UNINITIALIZED: u16 = 0;

/// Given the address and contents of a mint account, this function tells us what kind of metadata the account has.
/// It supports both spl-token and spl-token-2022 account layouts.
pub fn parse_token2022_metadata(
    mint_account: Pubkey,
    mint_account_data: &[u8],
) -> Result<MintMetadata, ParseError> {
    // mint accounts are at least 82 bytes long, so we reject
    if mint_account_data.len() < BASE_MINT_LEN {
        return Err(ParseError::NotAMintAccount);
    }

    // if exactly 82 bytes, it's either an spl token, or a token2022 with no extensions. this is valid
    if mint_account_data.len() == BASE_MINT_LEN {
        return Ok(MintMetadata::None); // no extensions
    }

    // if the mint has extensions, it's at least 166 bytes long. Anything in between is invalid.
    if mint_account_data.len() <= BASE_MINT_LEN_V2 {
        return Err(ParseError::NotAMintAccount);
    }

    // by this point, we know the mint has extensions. the layout here is:
    // 83..=165 padded with zeros (we verify here)
    if !mint_account_data[BASE_MINT_LEN..ACCOUNT_TYPE_OFFSET]
        .iter()
        .all(|&b| b == 0)
    {
        return Err(ParseError::NotAMintAccount);
    }

    // 166th byte is 1.
    let acct_type = *mint_account_data
        .get(ACCOUNT_TYPE_OFFSET)
        .ok_or(ParseError::UnexpectedEnd)?;
    if acct_type != ACCOUNT_TYPE_MINT {
        return Err(ParseError::NotMintAccountType);
    }

    // the rest is the extensions in TLV format
    let mut offset = ACCOUNT_TYPE_OFFSET + 1;
    let mut pointer: Option<MetadataPointer> = None;
    let mut meta: Option<TokenMetadata> = None;

    while offset + TLV_TYPE_SIZE + TLV_LEN_SIZE <= mint_account_data.len() {
        let t = u16::from_le_bytes([mint_account_data[offset], mint_account_data[offset + 1]]);
        let l = u16::from_le_bytes([mint_account_data[offset + 2], mint_account_data[offset + 3]]);
        offset += TLV_TYPE_SIZE + TLV_LEN_SIZE;

        // Bounds check for value
        let end = offset
            .checked_add(l as usize)
            .ok_or(ParseError::UnexpectedEnd)?;
        if end > mint_account_data.len() {
            return Err(ParseError::UnexpectedEnd);
        }
        let val = &mint_account_data[offset..end];
        offset = end;

        if t == EXT_UNINITIALIZED {
            // NOTE: this can actually never happen, see https://github.com/wormhole-foundation/wormhole/pull/4482#discussion_r2409540742.
            // We keep this code here as it was audited, and it doesn't actively hurt that much to have it.
            break; // padding
        }

        match t {
            EXT_METADATA_POINTER => {
                if val.len() != 64 {
                    return Err(ParseError::UnexpectedLength);
                }
                let authority = {
                    let a = Pubkey::from_slice(&val[0..32]).unwrap();
                    if a.is_zero() {
                        None
                    } else {
                        Some(a)
                    }
                };
                let metadata_address = Pubkey::from_slice(&val[32..64]).unwrap();
                pointer = Some(MetadataPointer {
                    authority,
                    metadata_address,
                });
            }
            EXT_TOKEN_METADATA => {
                // value layout:
                // [0..32)  update_authority (zero = None)
                // [32..64) mint
                // then: name: len(u32 LE) + bytes
                //       symbol: len + bytes
                //       uri: len + bytes
                //       kv_count: u32
                //       kv_count times: key(len+bytes), value(len+bytes)
                if val.len() < 64 {
                    return Err(ParseError::UnexpectedLength);
                }
                let update_authority = {
                    let a = Pubkey::from_slice(&val[0..32]).unwrap();
                    if a.is_zero() {
                        None
                    } else {
                        Some(a)
                    }
                };
                let mint = Pubkey::from_slice(&val[32..64]).unwrap();
                let mut cur = 64;

                fn read_u32_le(buf: &[u8], cur: &mut usize) -> Result<u32, ParseError> {
                    if *cur + 4 > buf.len() {
                        return Err(ParseError::UnexpectedEnd);
                    }
                    let n = u32::from_le_bytes([
                        buf[*cur],
                        buf[*cur + 1],
                        buf[*cur + 2],
                        buf[*cur + 3],
                    ]);
                    *cur += 4;
                    Ok(n)
                }
                fn read_string(buf: &[u8], cur: &mut usize) -> Result<String, ParseError> {
                    let len = read_u32_le(buf, cur)? as usize;
                    if *cur + len > buf.len() {
                        return Err(ParseError::UnexpectedEnd);
                    }
                    let s = std::str::from_utf8(&buf[*cur..*cur + len])
                        .map_err(|_| ParseError::InvalidUtf8)?;
                    *cur += len;
                    Ok(s.to_owned())
                }

                let name = read_string(val, &mut cur)?;
                let symbol = read_string(val, &mut cur)?;
                let uri = read_string(val, &mut cur)?;

                let kv_count = read_u32_le(val, &mut cur)? as usize;
                let mut additional_metadata = Vec::with_capacity(kv_count);
                for _ in 0..kv_count {
                    let k = read_string(val, &mut cur)?;
                    let v = read_string(val, &mut cur)?;
                    additional_metadata.push((k, v));
                }
                if cur != val.len() {
                    return Err(ParseError::LengthMismatch);
                }

                meta = Some(TokenMetadata {
                    update_authority,
                    mint,
                    name,
                    symbol,
                    uri,
                    additional_metadata,
                });
            }
            _ => {
                // Unknown extension; skip (we already advanced by its length)
            }
        }
    }

    match pointer {
        None => Ok(MintMetadata::None),
        Some(ptr) => {
            // Both pointer and embedded metadata exist
            if ptr.metadata_address == mint_account {
                if let Some(metadata) = meta {
                    // Pointer points to self, return embedded metadata
                    Ok(MintMetadata::Embedded(metadata))
                } else {
                    Err(ParseError::NoEmbeddedMetadata)
                }
            } else {
                // Pointer points elsewhere, return external pointer
                // NOTE: metadata may still be Some, but we ignore it (it's valid to have embedded metadata that's ignored)
                Ok(MintMetadata::External(ptr))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use solana_program::{
        program_option::COption,
        pubkey::Pubkey as SolanaPubkey,
    };
    use spl_pod::optional_keys::OptionalNonZeroPubkey;
    use spl_token_2022::{
        extension::{
            metadata_pointer::MetadataPointer,
            transfer_fee::TransferFeeConfig,
            BaseStateWithExtensionsMut,
            ExtensionType,
            StateWithExtensionsMut,
        },
        state::Mint,
    };
    use spl_token_metadata_interface::state::TokenMetadata;
    use std::str::FromStr;

    fn convert_pubkey(pk: SolanaPubkey) -> Pubkey {
        Pubkey::new(pk.to_bytes())
    }

    #[test]
    fn test_mint_with_metadata_pointer_and_metadata() {
        let mint_address = SolanaPubkey::new_unique();
        let authority = SolanaPubkey::new_unique();

        // Create TokenMetadata
        let token_metadata = TokenMetadata {
            update_authority: OptionalNonZeroPubkey(authority),
            mint: mint_address,
            name: "Test Token".to_string(),
            symbol: "TEST".to_string(),
            uri: "https://example.com/token.json".to_string(),
            additional_metadata: vec![],
        };

        // Calculate account size needed for both extensions
        let extension_types = vec![ExtensionType::MetadataPointer];
        let mut account_size =
            ExtensionType::try_calculate_account_len::<Mint>(&extension_types).unwrap();
        // Add space for variable-length TokenMetadata
        account_size += token_metadata.tlv_size_of().unwrap();

        // Create account data buffer
        let mut mint_data = vec![0; account_size];

        // Initialize the mint account with extensions
        let mut state =
            StateWithExtensionsMut::<Mint>::unpack_uninitialized(&mut mint_data).unwrap();

        // Set up base mint data
        state.base = Mint {
            mint_authority: COption::Some(authority),
            supply: 1000000,
            decimals: 6,
            is_initialized: true,
            freeze_authority: COption::None,
        };

        // Initialize MetadataPointer extension (pointing to self)
        let metadata_pointer_extension = state.init_extension::<MetadataPointer>(true).unwrap();
        metadata_pointer_extension.authority = OptionalNonZeroPubkey(authority);
        metadata_pointer_extension.metadata_address = OptionalNonZeroPubkey(mint_address);

        // Initialize TokenMetadata as a variable-length extension
        state
            .init_variable_len_extension(&token_metadata, true)
            .unwrap();

        // Pack the base state and initialize account type
        state.pack_base();
        state.init_account_type().unwrap();

        // Now test our parser against this properly serialized SPL Token-2022 data
        let result = parse_token2022_metadata(convert_pubkey(mint_address), &mint_data).unwrap();

        // Since metadata pointer points to self, we should get embedded metadata
        match &result {
            MintMetadata::Embedded(token_metadata) => {
                assert_eq!(
                    token_metadata.update_authority,
                    Some(convert_pubkey(authority))
                );
                assert_eq!(token_metadata.mint, convert_pubkey(mint_address));
                assert_eq!(token_metadata.name, "Test Token");
                assert_eq!(token_metadata.symbol, "TEST");
                assert_eq!(token_metadata.uri, "https://example.com/token.json");
                assert_eq!(token_metadata.additional_metadata.len(), 0);
            }
            _ => panic!("Expected embedded metadata"),
        }
    }

    /// Generate a random ExtensionType that has a known size.
    /// The only variable-length extension is TokenMetadata, which we skip.
    fn random_sized_extension() -> ExtensionType {
        use rand::Rng;
        use std::convert::TryFrom;
        loop {
            let ext_u16: u16 = rand::rng().random_range(0..=27);
            if let Ok(ext_type) = ExtensionType::try_from(ext_u16) {
                if ext_type != ExtensionType::TokenMetadata {
                    return ext_type;
                }
            }
        }
    }

    // In a previous version of this code, I misintepreted the mint account
    // extension layout, and thought that mint accounts can be _between_ 82
    // bytes (no extension) and 165 bytes (token account).
    //
    // It turns out that if the mint account has *any* extensions, it will
    // be padded out with 0s to 165, then a single 1 byte account type
    // discriminator is inserted, then the extensions follow.
    //
    // To verify this, we perform two tests:
    // 1. generate a random set of extensions and verify the calculated size is either 82 or > 165.
    // 2. create a mint account with a single small extension (MetadataPointer) and verify its size + the account discriminator.
    #[test]
    fn test_fuzz_mint_size() {
        use rand::Rng;

        let mut rng = rand::rng();
        for _ in 0..1000 {
            let len: usize = rng.random_range(0..=3);
            // now generate `len` random extensions from the ExtensionType enum.
            // ExtensionType is repr(u16), so we can generate a random u16 and
            // cast it to ExtensionType, then filter out invalid values.
            let mut extensions = Vec::new();
            for _ in 0..len {
                extensions.push(random_sized_extension());
            }
            let account_size =
                ExtensionType::try_calculate_account_len::<Mint>(&extensions).unwrap();
            assert!(account_size == 82 || account_size > 165);
        }
    }

    #[test]
    fn test_mint_with_small_extension() {
        // |------------------+--------------|
        // | field            | size (bytes) |
        // |------------------+--------------|
        // | base             |          165 |
        // | account type     |            1 |
        // | extension type   |            2 |
        // | extension length |            2 |
        // | metadata pointer |           64 |
        // |------------------+--------------|
        // | total            |          234 |

        let mint_address = SolanaPubkey::new_unique();
        let metadata_address = SolanaPubkey::new_unique();
        let authority = SolanaPubkey::new_unique();

        // Calculate account size needed for the extension
        let extension_types = vec![ExtensionType::MetadataPointer];
        let account_size =
            ExtensionType::try_calculate_account_len::<Mint>(&extension_types).unwrap();

        assert_eq!(account_size, 234);

        // Create account data buffer
        let mut mint_data = vec![0; account_size];

        // Initialize the mint account with extensions
        let mut state =
            StateWithExtensionsMut::<Mint>::unpack_uninitialized(&mut mint_data).unwrap();

        // Set up base mint data
        state.base = Mint {
            mint_authority: COption::Some(authority),
            supply: 1000000,
            decimals: 6,
            is_initialized: true,
            freeze_authority: COption::None,
        };

        // Initialize MetadataPointer extension (pointing to self)
        let metadata_pointer_extension = state.init_extension::<MetadataPointer>(true).unwrap();
        metadata_pointer_extension.authority = OptionalNonZeroPubkey(authority);
        metadata_pointer_extension.metadata_address = OptionalNonZeroPubkey(metadata_address);

        // Pack the base state and initialize account type
        state.pack_base();
        state.init_account_type().unwrap();

        let result = parse_token2022_metadata(convert_pubkey(mint_address), &mint_data).unwrap();

        // Since metadata pointer points elsewhere, we should get external metadata
        match &result {
            MintMetadata::External(addr) => {
                assert_eq!(addr.metadata_address, convert_pubkey(metadata_address));
            }
            _ => panic!("Expected external metadata"),
        }
    }

    #[test]
    fn test_basic_mint_no_extensions() {
        // Test basic 82-byte mint with no extensions
        let data = [0u8; 82];
        let dummy_mint = Pubkey::new([0u8; 32]);

        let result = parse_token2022_metadata(dummy_mint, &data).unwrap();
        match result {
            MintMetadata::None => {} // Expected
            _ => panic!("Expected no metadata"),
        }
    }

    #[test]
    fn test_error_cases() {
        let dummy_mint = Pubkey::new([0u8; 32]);

        // Test too short data
        let short_data = vec![0u8; 50];
        assert_eq!(
            parse_token2022_metadata(dummy_mint, &short_data),
            Err(ParseError::NotAMintAccount)
        );

        // Test wrong account length
        let wrong_type_data = vec![0u8; 100];
        assert_eq!(
            parse_token2022_metadata(dummy_mint, &wrong_type_data),
            Err(ParseError::NotAMintAccount)
        );
    }

    #[test]
    fn test_mint_with_multiple_extensions() {
        let mint_address = SolanaPubkey::new_unique();
        let authority = SolanaPubkey::new_unique();

        // Create TokenMetadata
        let token_metadata = TokenMetadata {
            update_authority: OptionalNonZeroPubkey(authority),
            mint: mint_address,
            name: "Multi-Extension Token".to_string(),
            symbol: "MULTI".to_string(),
            uri: "https://example.com/multi.json".to_string(),
            additional_metadata: vec![
                ("category".to_string(), "test".to_string()),
                ("version".to_string(), "1.0".to_string()),
            ],
        };

        // Calculate account size needed for multiple extensions
        let extension_types = vec![
            ExtensionType::MetadataPointer,
            ExtensionType::TransferFeeConfig,
        ];
        let mut account_size =
            ExtensionType::try_calculate_account_len::<Mint>(&extension_types).unwrap();
        // Add space for variable-length TokenMetadata
        account_size += token_metadata.tlv_size_of().unwrap();

        // Create account data buffer
        let mut mint_data = vec![0; account_size];

        // Initialize the mint account with extensions
        let mut state =
            StateWithExtensionsMut::<Mint>::unpack_uninitialized(&mut mint_data).unwrap();

        // Set up base mint data
        state.base = Mint {
            mint_authority: COption::Some(authority),
            supply: 5000000,
            decimals: 9,
            is_initialized: true,
            freeze_authority: COption::Some(authority),
        };

        // Initialize TransferFeeConfig extension (unrelated to metadata)
        let transfer_fee_extension = state.init_extension::<TransferFeeConfig>(true).unwrap();
        transfer_fee_extension.transfer_fee_config_authority = OptionalNonZeroPubkey(authority);
        transfer_fee_extension.withdraw_withheld_authority = OptionalNonZeroPubkey(authority);
        transfer_fee_extension.withheld_amount = 0.into();
        transfer_fee_extension
            .older_transfer_fee
            .transfer_fee_basis_points = 50.into(); // 0.5%
        transfer_fee_extension.older_transfer_fee.maximum_fee = 1000000.into(); // 1 token
        transfer_fee_extension
            .newer_transfer_fee
            .transfer_fee_basis_points = 25.into(); // 0.25%
        transfer_fee_extension.newer_transfer_fee.maximum_fee = 500000.into(); // 0.5 token

        // Initialize MetadataPointer extension (pointing to self)
        let metadata_pointer_extension = state.init_extension::<MetadataPointer>(true).unwrap();
        metadata_pointer_extension.authority = OptionalNonZeroPubkey(authority);
        metadata_pointer_extension.metadata_address = OptionalNonZeroPubkey(mint_address);

        // Initialize TokenMetadata as a variable-length extension
        state
            .init_variable_len_extension(&token_metadata, true)
            .unwrap();

        // Pack the base state and initialize account type
        state.pack_base();
        state.init_account_type().unwrap();

        // Now test our parser against this multi-extension data
        let result = parse_token2022_metadata(convert_pubkey(mint_address), &mint_data).unwrap();

        // Since metadata pointer points to self, we should get embedded metadata (should work despite other extensions)
        match &result {
            MintMetadata::Embedded(token_metadata) => {
                assert_eq!(
                    token_metadata.update_authority,
                    Some(convert_pubkey(authority))
                );
                assert_eq!(token_metadata.mint, convert_pubkey(mint_address));
                assert_eq!(token_metadata.name, "Multi-Extension Token");
                assert_eq!(token_metadata.symbol, "MULTI");
                assert_eq!(token_metadata.uri, "https://example.com/multi.json");
                assert_eq!(token_metadata.additional_metadata.len(), 2);
                assert_eq!(
                    token_metadata.additional_metadata[0],
                    ("category".to_string(), "test".to_string())
                );
                assert_eq!(
                    token_metadata.additional_metadata[1],
                    ("version".to_string(), "1.0".to_string())
                );
            }
            _ => panic!("Expected embedded metadata despite multiple extensions"),
        }
    }

    #[test]
    fn test_real_world_pyusd_mint_data() {
        // Real-world PYUSD mint account data from mainnet: 2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo
        // This mint has both a metadata pointer and embedded TokenMetadata
        let hex_data = "01000000dd4c486c90f8b6f007c304ef2481f805186be8fd5f52acd1025cb79b9f67ff216c9d1390e1cd000006010100000017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010300200017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b0c00200017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b01006c0017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b17853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b00000000000000005d02000000000000000000000000000000005d02000000000000000000000000000000000400410017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b0000000000000000000000000000000000000000000000000000000000000000001000810017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b1c37e6433b7304dd82737ae40d9b8bf3c49f5b0e6c49a8d53328b3e506901c5701000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e00400017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b00000000000000000000000000000000000000000000000000000000000000001200400017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b1792483b6c8a2a87b7471d814f9591f9395c840a9ce3d9f4d5ba7d3a4b8a749e1300ae0017853261ef6ab8532a67f053865aad31293fcf07cf120ab5b9a15706548dc02b1792483b6c8a2a87b7471d814f9591f9395c840a9ce3d9f4d5ba7d3a4b8a749e0a00000050617950616c205553440500000050595553444f00000068747470733a2f2f746f6b656e2d6d657461646174612e7061786f732e636f6d2f70797573645f6d657461646174612f70726f642f736f6c616e612f70797573645f6d657461646174612e6a736f6e00000000";

        let raw_data = hex::decode(hex_data).expect("Valid hex string");

        let pyusd_mint = convert_pubkey(
            SolanaPubkey::from_str("2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo").unwrap(),
        );

        // Parse using our custom parser
        let result = parse_token2022_metadata(pyusd_mint, &raw_data).unwrap();

        let metadata_address = convert_pubkey(
            SolanaPubkey::from_str("2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo").unwrap(),
        );

        match &result {
            MintMetadata::Embedded(token_metadata) => {
                // PYUSD has embedded metadata (metadata pointer points to self)
                assert_eq!(token_metadata.mint, metadata_address);
                assert_eq!(token_metadata.name, "PayPal USD");
                assert_eq!(token_metadata.symbol, "PYUSD");
                assert_eq!(
                    token_metadata.uri,
                    "https://token-metadata.paxos.com/pyusd_metadata/prod/solana/pyusd_metadata.json"
                );
                assert_eq!(token_metadata.additional_metadata.len(), 0);
            }
            _ => panic!("Expected embedded metadata for PYUSD (pointer points to mint)"),
        }
    }

    #[test]
    fn test_real_world_usdc_mint_data() {
        // Real-world USDC mint account data from mainnet: EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
        // This is a regular spl token. We want to make sure we can parse it still.
        let hex_data = "0100000098fe86e88d9be2ea8bc1cca4878b2988c240f52b8424bfb40ed1a2ddcb5e199b3e8ef2b0d02020000601010000006270aa8a59c59405b45286c86772e6cd126e9b8a5d3a38536d37f7b414e8b667";

        let raw_data = hex::decode(hex_data).expect("Valid hex string");
        let dummy_mint = Pubkey::new([0u8; 32]); // USDC mint address not needed for this test

        // Parse using our custom parser
        let result = parse_token2022_metadata(dummy_mint, &raw_data).unwrap();

        // Should return None since this is a basic SPL token with no extensions
        match result {
            MintMetadata::None => {} // Expected
            _ => panic!("Expected no metadata for basic SPL token"),
        }
    }

    #[test]
    fn test_real_world_wsol_token2022_mint_data() {
        // This is a token2022 token with no extensions: 9pan9bMn5HatX4EJdBwg9VgCa7Uz5HL8N1m5D3NdXejP
        // We make sure we can parse its mint account the same
        let hex_data = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000901000000000000000000000000000000000000000000000000000000000000000000000000";

        let raw_data = hex::decode(hex_data).expect("Valid hex string");
        let dummy_mint = Pubkey::new([0u8; 32]); // Mint address not needed for this test

        // Parse using our custom parser
        let result = parse_token2022_metadata(dummy_mint, &raw_data).unwrap();

        // Should return None since this Token-2022 mint has no extensions
        match result {
            MintMetadata::None => {} // Expected
            _ => panic!("Expected no metadata for Token-2022 mint with no extensions"),
        }
    }
}
