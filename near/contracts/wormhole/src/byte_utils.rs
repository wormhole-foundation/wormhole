pub trait ByteUtils {
    fn get_u8(&self, index: usize) -> u8;
    fn get_u16(&self, index: usize) -> u16;
    fn get_u32(&self, index: usize) -> u32;
    fn get_u64(&self, index: usize) -> u64;

    fn get_u128_be(&self, index: usize) -> u128;
    /// High 128 then low 128
    fn get_u256(&self, index: usize) -> (u128, u128);
    //    fn get_address(&self, index: usize) -> CanonicalAddr;
    fn get_bytes32(&self, index: usize) -> &[u8];
    fn get_bytes(&self, index: usize, bytes: usize) -> &[u8];
    fn get_const_bytes<const N: usize>(&self, index: usize) -> [u8; N];
}

impl ByteUtils for &[u8] {
    fn get_u8(&self, index: usize) -> u8 {
        self[index]
    }
    fn get_u16(&self, index: usize) -> u16 {
        let mut bytes: [u8; 16 / 8] = [0; 16 / 8];
        bytes.copy_from_slice(&self[index..index + 2]);
        u16::from_be_bytes(bytes)
    }
    fn get_u32(&self, index: usize) -> u32 {
        let mut bytes: [u8; 32 / 8] = [0; 32 / 8];
        bytes.copy_from_slice(&self[index..index + 4]);
        u32::from_be_bytes(bytes)
    }
    fn get_u64(&self, index: usize) -> u64 {
        let mut bytes: [u8; 64 / 8] = [0; 64 / 8];
        bytes.copy_from_slice(&self[index..index + 8]);
        u64::from_be_bytes(bytes)
    }
    fn get_u128_be(&self, index: usize) -> u128 {
        let mut bytes: [u8; 128 / 8] = [0; 128 / 8];
        bytes.copy_from_slice(&self[index..index + 128 / 8]);
        u128::from_be_bytes(bytes)
    }
    fn get_u256(&self, index: usize) -> (u128, u128) {
        (self.get_u128_be(index), self.get_u128_be(index + 128 / 8))
    }
    //    fn get_address(&self, index: usize) -> CanonicalAddr {
    // 32 bytes are reserved for addresses, but only the last 20 bytes are taken by the actual address
    //        CanonicalAddr::from(&self[index + 32 - 20..index + 32])
    //    }
    fn get_bytes32(&self, index: usize) -> &[u8] {
        &self[index..index + 32]
    }

    fn get_bytes(&self, index: usize, bytes: usize) -> &[u8] {
        &self[index..index + bytes]
    }

    fn get_const_bytes<const N: usize>(&self, index: usize) -> [u8; N] {
        let mut bytes: [u8; N] = [0; N];
        bytes.copy_from_slice(&self[index..index + N]);
        bytes
    }
}

/// Left-pad a 20 byte address with 0s
//pub fn extend_address_to_32(addr: &CanonicalAddr) -> Vec<u8> {
//    extend_address_to_32_array(addr).to_vec()
//}

//pub fn extend_address_to_32_array(addr: &CanonicalAddr) -> [u8; 32] {
//    let mut v: Vec<u8> = vec![0; 12];
//    v.extend(addr.as_slice());
//    let mut result: [u8; 32] = [0; 32];
//    result.copy_from_slice(&v);
//    result
//}

/// Turn a string into a fixed length array. If the string is shorter than the
/// resulting array, it gets padded with \0s on the right. If longer, it gets
/// truncated.
pub fn string_to_array<const N: usize>(s: &str) -> [u8; N] {
    let bytes = s.as_bytes();
    let len = usize::min(N, bytes.len());
    let zeros = vec![0; N - len];
    let padded = [bytes[..len].to_vec(), zeros].concat();
    let mut result: [u8; N] = [0; N];
    result.copy_from_slice(&padded);
    result
}

pub fn extend_string_to_32(s: &str) -> Vec<u8> {
    string_to_array::<32>(s).to_vec()
}

pub fn get_string_from_32(v: &[u8]) -> String {
    let s = String::from_utf8_lossy(v);
    s.chars().filter(|c| c != &'\0').collect()
}
