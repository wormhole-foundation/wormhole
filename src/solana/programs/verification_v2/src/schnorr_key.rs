use anchor_lang::prelude::{
  AccountInfo,
  AnchorDeserialize,
  AnchorSerialize,
  Clock,
  Discriminator,
  InitSpace,
  Key,
  Program,
  Pubkey,
  Result,
  SolanaSysvar,
  Space,
  System,
  account,
  borsh,
  error_code,
};
#[cfg(feature = "idl-build")]
use anchor_lang::{
  IdlBuild,
  idl::types::{
    IdlArrayLen,
    IdlDefinedFields,
    IdlField,
    IdlSerialization,
    IdlType,
    IdlTypeDef,
    IdlTypeDefTy,
  },
};
use anchor_lang::solana_program::{keccak::{Hash, hash}, secp256k1_recover};
use primitive_types::{U256, U512};
use std::io::{Read, Write};
use std::ops::{Shr, Sub};

use crate::hex_literal::hex;
use crate::vaa::VAASchnorrSignature;
use crate::utils::{init_account, SeedPrefix};
use crate::ID;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SchnorrKey {
  pub key: U256,
}

#[cfg(feature = "idl-build")]
impl IdlBuild for SchnorrKey {
  fn create_type() -> Option<IdlTypeDef> {
    Some(IdlTypeDef {
      name: "SchnorrKey".to_string(),
      docs: vec![],
      serialization: IdlSerialization::Borsh,
      repr: None,
      generics: vec![],
      ty: IdlTypeDefTy::Struct {
        fields: Some(IdlDefinedFields::Named(vec![
          IdlField {
            name: "key".to_string(),
            docs: vec![],
            ty: IdlType::Array(Box::new(IdlType::U8), IdlArrayLen::Value(32)),
          },
        ])),
      },
    })
  }
}

#[error_code]
pub enum SchnorrKeyError {
    #[msg("Signature does not satisfy preconditions")]
    InvalidSignature,
    SignatureVerificationFailed,
}

impl SchnorrKey {
  // This is only used to validate when appending a pubkey
  // so we don't really care about its representation.
  const HALF_Q: [u8; 32] = hex!("7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0");

  // The following constants are used during verification.
  // We chose the representation that makes verification cheaper.
  // Concretely, these are arrays of 64 bit integers where the least significative parts come first.
  // See the math_tests module at the end of this file for tests related to these.

  // Q is the curve order of secp256k1
  const Q_U256: [u64; 4] = [
    0xBFD25E8CD0364141,
    0xBAAEDCE6AF48A03B,
    0xFFFFFFFFFFFFFFFE,
    0xFFFFFFFFFFFFFFFF
  ];
  // Used to approximate division by Q
  // floor(2^511 / Q)
  const ΜQ_U256: [u64; 4] = [
    0x2016d0b997e4df60,
    0xa2a8918ca85bafe2,
    0x0000000000000000,
    0x8000000000000000
  ];

  pub fn q() -> U256 {
    U256(SchnorrKey::Q_U256)
  }

  pub fn μq() -> U256 {
    U256(SchnorrKey::ΜQ_U256)
  }

  pub fn half_q() -> U256 {
    U256::from_big_endian(&SchnorrKey::HALF_Q)
  }

  pub fn px(&self) -> U256 {
    self.key.shr(U256::one())
  }

  pub fn parity(&self) -> bool {
    self.key.bit(0)
  }

  pub fn is_valid(&self) -> bool {
    let px = self.px();
    !px.is_zero() && px.le(&Self::half_q())
  }

  #[inline(always)]
  pub fn check_signature(&self, message_hash: &Hash, signature: &VAASchnorrSignature) -> Result<()> {
    let px = self.px();
    let parity = self.parity();
    let q = Self::q();
    let r = signature.r;
    let s = signature.s;

    // Calculate the message challenge
    let mut hash_bytes = [0u8; 85];
    hash_bytes[0..32].copy_from_slice(&px.to_big_endian());
    hash_bytes[32] = parity as u8;
    hash_bytes[33..65].copy_from_slice(&message_hash.to_bytes());
    hash_bytes[65..85].copy_from_slice(&r);

    let e = U256::from_big_endian(&hash(&hash_bytes).to_bytes());

    // Calculate the recovery inputs
    // Barrett reductions work as long as the product a*b doesn't exceed Q^2.
    // Here, one of the factors is the x component of the public key.
    // In particular, we already know that px <= Q/2
    // See SchnorrKey::is_valid() and math_tests::test_mulmod_upper_bound_is_safe()
    let sp = q.sub(Self::mulmod_barrett_q(s, px));
    let ep = Self::mulmod_barrett_q(e, px);

    if sp.is_zero() || ep.is_zero() {
      return Err(SchnorrKeyError::InvalidSignature.into());
    }

    // Prepare the ecrecover inputs
    let mut signature_bytes = [0u8; 64];
    // this is r
    signature_bytes[0..32].copy_from_slice(&px.to_big_endian());
    // this is s
    signature_bytes[32..64].copy_from_slice(&ep.to_big_endian());
    let sp_buf = sp.to_big_endian();

    let recovered_pubkey = secp256k1_recover::secp256k1_recover(
      &sp_buf,
      parity as u8,
      &signature_bytes
    ).unwrap();

    let recovered_address = &hash(&recovered_pubkey.to_bytes()).to_bytes()[12..];

    if recovered_address != r {
      return Err(SchnorrKeyError::SignatureVerificationFailed.into());
    }

    Ok(())
  }

  /// This implements the modulo step via barrett reduction.
  /// r = ab - floor(floor(ab / 2^256) * μq / 2^255) * q
  /// where q = Q, the secp256k1 curve order
  ///       ab = a*b, the product of the inputs to mulmod
  ///       μq = floor(2^511 / q), used to approximate division by q
  ///       r, i.e. the result: representant of a*b mod q
  /// Note that the scaling factor was chosen so that μq fits into 256 bits.
  #[inline(always)]
  fn mulmod_barrett_q(a: U256, b: U256) -> U256 {
    let ab = a.full_mul(b);

    // ab_high = floor(ab / 2^256)   → top 256 bits
    let ab_high: [u64; 4] = ab.0[4..8].try_into().unwrap();

    // t1 = ab_high * μQ
    let t1 = U256(ab_high).full_mul(SchnorrKey::μq());

    // t2 = floor(t1 / 2^255)        → top 257 bits
    // but (t1 >> 255) fits in 256 bits because:
    // ab fits in 511 bits => top 256 bits have the most significant bit cleared
    // then ab_high fits in 255 bits => ab_high * μq, i.e. t1, fits in 511 bits
    let t2 = U256((t1 >> 255).0[0..4].try_into().unwrap());

    let q = SchnorrKey::q();
    let representative = ab - t2.full_mul(q);

    // representative should be in [0, 3Q), so we subtract Q if needed
    let q_u512 = U512::from(q);
    let mut result = representative;
    if result >= q_u512 {
      result -= q_u512;
      if result >= q_u512 {
        result -= q_u512;
      }
    }

    result.try_into().unwrap()
  }
}

impl Space for SchnorrKey {
  const INIT_SPACE: usize = 32;
}

impl AnchorSerialize for SchnorrKey {
  fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
    if !self.is_valid() {
      return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "Invalid schnorr key"));
    }

    writer.write_all(&self.key.to_big_endian())?;
    Ok(())
  }
}

impl AnchorDeserialize for SchnorrKey {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
    let mut key_buf = [0u8; 32];
    reader.read_exact(&mut key_buf)?;
    let key = SchnorrKey { key: U256::from_big_endian(&key_buf) };

    Ok(key)
  }
}


#[account]
#[derive(InitSpace)]
pub struct SchnorrKeyAccount {
  pub index: u32,
  pub schnorr_key: SchnorrKey,
  pub expiration_timestamp: u64,
}

impl SchnorrKeyAccount {
  pub fn is_unexpired(&self) -> bool {
    self.expiration_timestamp == 0 || self.expiration_timestamp > Clock::get().unwrap().unix_timestamp as u64
  }

  pub fn update_expiration_timestamp(&mut self, time_lapse: u64) {
    let current_timestamp = Clock::get().unwrap().unix_timestamp as u64;
    self.expiration_timestamp = current_timestamp + time_lapse;
  }
}

impl SeedPrefix for SchnorrKeyAccount {
  const SEED_PREFIX: &'static [u8] = b"schnorrkey";
}

pub fn init_schnorr_key_account<'info>(
  new_schnorr_key: AccountInfo<'info>,
  schnorr_key_index: u32,
  schnorr_key: SchnorrKey,
  system_program: &Program<'info, System>,
  payer: AccountInfo<'info>
) -> Result<()> {
  // We need to parse the schnorr key append VAA payload
  // to perform the derivation.
  // This is why the initialization happens manually here.

  let (pubkey, bump) = Pubkey::find_program_address(
    &[SchnorrKeyAccount::SEED_PREFIX, &schnorr_key_index.to_le_bytes()],
    &ID,
  );

  if pubkey != new_schnorr_key.key() {
    return Err(AppendSchnorrKeyError::AccountMismatchSchnorrKeyIndex.into());
  }

  let schnorr_key_seeds = [SchnorrKeyAccount::SEED_PREFIX, &schnorr_key_index.to_le_bytes(), &[bump]];

  init_account(
    new_schnorr_key.clone(),
    &schnorr_key_seeds,
    &system_program,
    payer,
    SchnorrKeyAccount{
      index: schnorr_key_index,
      schnorr_key,
      expiration_timestamp: 0,
    }
  )?;

  Ok(())
}


#[error_code]
pub enum AppendSchnorrKeyError {
  InvalidVAA,
  InvalidGovernanceChainId,
  InvalidGovernanceAddress,
  #[msg("New key must have strictly greater index")]
  InvalidNewKeyIndex,
  #[msg("Old schnorr key must be the latest key")]
  InvalidOldSchnorrKey,
  #[msg("Schnorr key address mismatches Schnorr key index")]
  AccountMismatchSchnorrKeyIndex,
}

#[cfg(test)]
mod math_tests {
  use super::{SchnorrKey, U256, U512, Shr};
  use num_bigint::BigUint;
  use num_traits::{Num, One};
  use rstest::rstest;
  use proptest::prelude::{proptest, any, Strategy, ProptestConfig};

  #[test]
  fn q_is_correct() {
    let q = U256::from_str_radix(
      "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
      16
    ).unwrap();
    assert_eq!(SchnorrKey::q(), q);
  }

  #[test]
  fn half_q_is_correct() {
    assert_eq!(SchnorrKey::q().shr(U256::one()), SchnorrKey::half_q());
  }

  #[test]
  fn μq_is_correct() {
    let q = BigUint::from_str_radix(
      "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
      16
    ).unwrap();

    // 2^511
    let two_exp511 = BigUint::one() << 511;

    // μ = floor(2^511 / Q)
    let mu: BigUint = &two_exp511 / &q;
    let μq = U256::from_little_endian(&mu.to_bytes_le());

    assert_eq!(μq, SchnorrKey::μq());
  }

  #[rstest]
  #[case("FFEFFFF13FFFFF6FF8FF9FFEBAAEDCE6AF48A03BBFD25E8CD0364100",
         "ABEFFFF13FFFFF6FF8FF9FFEBAAEDCE6AF48A03BBFD25E341"                )]
  #[case("CDFF13FFFB6FF8FF9FFEBAAEDCE6AF48A03BBFD25E80000000000",
         "486ABBA13FFFFF6FF8FF9FFEBAAEDCE6AF48A03B324365316"                )]
  #[case("1",
         "ABEF00F13FFFFF6FF8FF9FFEBAAEDCE6AF48A03BBFD25E341"                )]
  #[case("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
         "7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B209F" )]
  fn test_mulmod(#[case] a_str: &str, #[case] b_str: &str) {
    let a = U256::from_str_radix(a_str, 16).unwrap();
    let b = U256::from_str_radix(b_str, 16).unwrap();

    let mulmod = SchnorrKey::mulmod_barrett_q(a, b);
    let expected: U256 = (a.full_mul(b) % U512::from(SchnorrKey::q())).try_into().unwrap();
    assert_eq!(mulmod, expected);
  }

  #[test]
  fn test_mulmod_upper_bound_is_safe() {
    let a = U256::max_value();
    // We know that one of the factors is at most half Q (see SchnorrKey::is_valid)
    let b = SchnorrKey::half_q();

    let q = SchnorrKey::q();

    let q_2 = q.full_mul(q);
    assert!(a.full_mul(b) < q_2);

    let mulmod = SchnorrKey::mulmod_barrett_q(a, b);

    let expected: U256 = (a.full_mul(b) % U512::from(q)).try_into().unwrap();
    assert_eq!(mulmod, expected);
  }

  fn u256_strategy() -> impl Strategy<Value = U256> {
    any::<[u64; 4]>().prop_map(|words| U256(words))
  }

  proptest! {
    #![proptest_config(ProptestConfig::with_cases(10000))]

    #[test]
    fn test_mulmod_random_inputs(a in u256_strategy(), b in u256_strategy()) {
      let q = SchnorrKey::q();

      let mulmod = SchnorrKey::mulmod_barrett_q(a, b);
      let expected: U256 = (a.full_mul(b) % U512::from(q)).try_into().unwrap();
      assert_eq!(mulmod, expected);
    }
  }
}
