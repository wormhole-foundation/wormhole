pub extern crate secp256k1;

use std::io::Write;

use hex;
use num_traits::AsPrimitive;
use secp256k1::curve::{Affine, ECMULT_CONTEXT, ECMULT_GEN_CONTEXT, ECMultGenContext, Field, Jacobian, Scalar};
use sha3::Digest;

enum SchnorrError {
    InvalidPubKey
}

struct PublicKey {
    x: Field,
    y: Field,
}

struct SchnorrSignature {
    address: [u8; 20],
    signature: [u8; 32],
}

impl PublicKey {
    pub fn deserialize(d: &[u8; 64]) -> PublicKey {
        let mut x = Field::default();
        let mut y = Field::default();
        x.set_b32(array_ref![d,0,32]);
        y.set_b32(array_ref![d,32,32]);
        x.normalize();
        y.normalize();
        return PublicKey {
            x,
            y,
        };
    }

    pub fn verify(&self) -> bool {
        //TODO optimize
        return self.x < Field::new(
            0x7fffffff, 0xffffffff, 0xffffffff, 0xffffffff,
            0x5d576e73, 0x57a4501d, 0xdfe92f46, 0x681b20a1,
        );
    }
}

impl SchnorrSignature {
    pub fn verify_signature(&self, msg: &[u8; 32], pub_key: &PublicKey) -> Result<bool, SchnorrError> {
        let mut af = Affine::default();
        af.set_xy(&pub_key.x, &pub_key.y);
        let mut pubkey_j: Jacobian = Jacobian::default();
        pubkey_j.set_ge(&af);

        // Verify that the pubkey is a curve point
        let mut elem = Affine::default();
        if !elem.set_xo_var(&pub_key.x, pub_key.y.is_odd()) {
            return Err(SchnorrError::InvalidPubKey);
        }
        elem.y.normalize();

        // Make sure that the ordinates are equal
        if elem.y.b32() != pub_key.y.b32() {
            return Err(SchnorrError::InvalidPubKey);
        }

        // Generate the challenge
        let mut h = sha3::Keccak256::default();
        h.write(&pub_key.x.b32()); // pub key x coordinate
        h.write(&[(pub_key.y.is_odd()).as_()]); // y parity
        h.write(msg); // msg
        h.write(&self.address); // nonceTimesGeneratorAddress
        let challenge = h.finalize();

        let mut e = Scalar::default();
        e.set_b32(array_ref![challenge,0,32]);

        let mut s = Scalar::default();
        s.set_b32(&self.signature);

        // Calculate s x G + e x P
        let mut k: Jacobian = Jacobian::default();
        ECMULT_CONTEXT.ecmult(&mut k, &pubkey_j, &e, &s);

        let r = jacobian_to_normalized_affine(&k);

        // Generate Ethereum address from calculated point
        let eth_addr = affine_to_eth_address(&r);

        // Verify that addr(k) == sig.address
        Ok(eth_addr == self.address)
    }
}

fn affine_to_eth_address(a: &Affine) -> [u8; 20] {
    let mut h = sha3::Keccak256::default();
    h.write(a.x.b32().as_ref()); // result key x coordinate
    h.write(a.y.b32().as_ref()); // result key y coordinate

    let out = h.finalize();
    let mut out_addr = [0; 20];
    out_addr.copy_from_slice(&out[12..]);

    out_addr
}

fn jacobian_to_normalized_affine(j: &Jacobian) -> Affine {
    let mut r = Affine::default();
    r.set_gej(j);
    r.x.normalize();
    r.y.normalize();

    r
}

#[cfg(test)]
mod tests {
    use std::io::Write;

    use hex;
    use secp256k1;
    use secp256k1::curve::{ECMULT_CONTEXT, ECMULT_GEN_CONTEXT};

    use crate::crypto::{PublicKey, SchnorrSignature};

    #[test]
    fn verify_signature() {
        let msggg = hex::decode("0194fdc2fa2ffcc041d3ff12045b73c86e4ff95ff662a5eee82abdf44a2d0b75").unwrap();
        let siggg = hex::decode("ee5884a66454baca985f4453c05394214a75dc38956ea39f12cc429f081aae4b").unwrap();
        let addr = hex::decode("9addd8a38fea7e1b94550e5bc249309a633dfa63").unwrap();
        let pb = hex::decode("ae92ce7553993f04400c6976f8cd4540ae076bf0131eec8b35ae0ff9fc577a901de834d0f62ae6ecbeec2124595b06bce078b8133b4dda3855cf346feb2b2ca2").unwrap();

        let sig = SchnorrSignature {
            address: *array_ref![addr,0,20],
            signature: *array_ref![siggg,0,32],
        };

        let pub_key = PublicKey::deserialize(array_ref![pb,0,64]);
        assert!(pub_key.verify(), "invalid public key");

        let msg = array_ref![msggg,0,32];
        match sig.verify_signature(msg, &pub_key) {
            Ok(res) => assert!(res, "signature should be valid"),
            Err(err) => assert!(false, "signature verification failed")
        }
    }
}
