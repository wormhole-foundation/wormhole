use dep::aztec::{protocol_types::traits::{Deserialize, Packable, Serialize}, prelude::AztecAddress};
use std::meta::derive;

#[derive(Deserialize, Packable, Serialize)]
pub struct Provider {
    pub chain_id: u16,
    pub evm_chain_id: u16,
}

// Keep your Guardian_PK EXACTLY as is - don't touch it
#[derive(Deserialize, Packable, Serialize)]
pub struct Guardian_PK {
    // Address (20 bytes)
    pub value0: u8, pub value1: u8, pub value2: u8, pub value3: u8, pub value4: u8,
    pub value5: u8, pub value6: u8, pub value7: u8, pub value8: u8, pub value9: u8,
    pub value10: u8, pub value11: u8, pub value12: u8, pub value13: u8, pub value14: u8,
    pub value15: u8, pub value16: u8, pub value17: u8, pub value18: u8, pub value19: u8,
    
    // Public Key X (32 bytes)
    pub pub_key_x0: u8, pub pub_key_x1: u8, pub pub_key_x2: u8, pub pub_key_x3: u8,
    pub pub_key_x4: u8, pub pub_key_x5: u8, pub pub_key_x6: u8, pub pub_key_x7: u8,
    pub pub_key_x8: u8, pub pub_key_x9: u8, pub pub_key_x10: u8, pub pub_key_x11: u8,
    pub pub_key_x12: u8, pub pub_key_x13: u8, pub pub_key_x14: u8, pub pub_key_x15: u8,
    pub pub_key_x16: u8, pub pub_key_x17: u8, pub pub_key_x18: u8, pub pub_key_x19: u8,
    pub pub_key_x20: u8, pub pub_key_x21: u8, pub pub_key_x22: u8, pub pub_key_x23: u8,
    pub pub_key_x24: u8, pub pub_key_x25: u8, pub pub_key_x26: u8, pub pub_key_x27: u8,
    pub pub_key_x28: u8, pub pub_key_x29: u8, pub pub_key_x30: u8, pub pub_key_x31: u8,
    
    // Public Key Y (32 bytes)
    pub pub_key_y0: u8, pub pub_key_y1: u8, pub pub_key_y2: u8, pub pub_key_y3: u8,
    pub pub_key_y4: u8, pub pub_key_y5: u8, pub pub_key_y6: u8, pub pub_key_y7: u8,
    pub pub_key_y8: u8, pub pub_key_y9: u8, pub pub_key_y10: u8, pub pub_key_y11: u8,
    pub pub_key_y12: u8, pub pub_key_y13: u8, pub pub_key_y14: u8, pub pub_key_y15: u8,
    pub pub_key_y16: u8, pub pub_key_y17: u8, pub pub_key_y18: u8, pub pub_key_y19: u8,
    pub pub_key_y20: u8, pub pub_key_y21: u8, pub pub_key_y22: u8, pub pub_key_y23: u8,
    pub pub_key_y24: u8, pub pub_key_y25: u8, pub pub_key_y26: u8, pub pub_key_y27: u8,
    pub pub_key_y28: u8, pub pub_key_y29: u8, pub pub_key_y30: u8, pub pub_key_y31: u8,
}

impl Guardian_PK {
    pub fn new(address: [u8; 20], pub_key_x: [u8; 32], pub_key_y: [u8; 32]) -> Self {
        Guardian_PK { 
            // Address
            value0: address[0], value1: address[1], value2: address[2], value3: address[3], value4: address[4],
            value5: address[5], value6: address[6], value7: address[7], value8: address[8], value9: address[9],
            value10: address[10], value11: address[11], value12: address[12], value13: address[13], value14: address[14],
            value15: address[15], value16: address[16], value17: address[17], value18: address[18], value19: address[19],
            
            // Public Key X
            pub_key_x0: pub_key_x[0], pub_key_x1: pub_key_x[1], pub_key_x2: pub_key_x[2], pub_key_x3: pub_key_x[3],
            pub_key_x4: pub_key_x[4], pub_key_x5: pub_key_x[5], pub_key_x6: pub_key_x[6], pub_key_x7: pub_key_x[7],
            pub_key_x8: pub_key_x[8], pub_key_x9: pub_key_x[9], pub_key_x10: pub_key_x[10], pub_key_x11: pub_key_x[11],
            pub_key_x12: pub_key_x[12], pub_key_x13: pub_key_x[13], pub_key_x14: pub_key_x[14], pub_key_x15: pub_key_x[15],
            pub_key_x16: pub_key_x[16], pub_key_x17: pub_key_x[17], pub_key_x18: pub_key_x[18], pub_key_x19: pub_key_x[19],
            pub_key_x20: pub_key_x[20], pub_key_x21: pub_key_x[21], pub_key_x22: pub_key_x[22], pub_key_x23: pub_key_x[23],
            pub_key_x24: pub_key_x[24], pub_key_x25: pub_key_x[25], pub_key_x26: pub_key_x[26], pub_key_x27: pub_key_x[27],
            pub_key_x28: pub_key_x[28], pub_key_x29: pub_key_x[29], pub_key_x30: pub_key_x[30], pub_key_x31: pub_key_x[31],
            
            // Public Key Y
            pub_key_y0: pub_key_y[0], pub_key_y1: pub_key_y[1], pub_key_y2: pub_key_y[2], pub_key_y3: pub_key_y[3],
            pub_key_y4: pub_key_y[4], pub_key_y5: pub_key_y[5], pub_key_y6: pub_key_y[6], pub_key_y7: pub_key_y[7],
            pub_key_y8: pub_key_y[8], pub_key_y9: pub_key_y[9], pub_key_y10: pub_key_y[10], pub_key_y11: pub_key_y[11],
            pub_key_y12: pub_key_y[12], pub_key_y13: pub_key_y[13], pub_key_y14: pub_key_y[14], pub_key_y15: pub_key_y[15],
            pub_key_y16: pub_key_y[16], pub_key_y17: pub_key_y[17], pub_key_y18: pub_key_y[18], pub_key_y19: pub_key_y[19],
            pub_key_y20: pub_key_y[20], pub_key_y21: pub_key_y[21], pub_key_y22: pub_key_y[22], pub_key_y23: pub_key_y[23],
            pub_key_y24: pub_key_y[24], pub_key_y25: pub_key_y[25], pub_key_y26: pub_key_y[26], pub_key_y27: pub_key_y[27],
            pub_key_y28: pub_key_y[28], pub_key_y29: pub_key_y[29], pub_key_y30: pub_key_y[30], pub_key_y31: pub_key_y[31],
        }
    }

    pub fn default() -> Self {
        Guardian_PK::new([0; 20], [0; 32], [0; 32])
    }

    pub fn as_array(Guardian_PK: Guardian_PK) -> [u8; 20] {
        [Guardian_PK.value0, Guardian_PK.value1, Guardian_PK.value2, Guardian_PK.value3, Guardian_PK.value4, 
         Guardian_PK.value5, Guardian_PK.value6, Guardian_PK.value7, Guardian_PK.value8, Guardian_PK.value9,
         Guardian_PK.value10, Guardian_PK.value11, Guardian_PK.value12, Guardian_PK.value13, Guardian_PK.value14,
         Guardian_PK.value15, Guardian_PK.value16, Guardian_PK.value17, Guardian_PK.value18, Guardian_PK.value19]
    }

    pub fn get_pub_key_x(self) -> [u8; 32] {
        [self.pub_key_x0, self.pub_key_x1, self.pub_key_x2, self.pub_key_x3,
         self.pub_key_x4, self.pub_key_x5, self.pub_key_x6, self.pub_key_x7,
         self.pub_key_x8, self.pub_key_x9, self.pub_key_x10, self.pub_key_x11,
         self.pub_key_x12, self.pub_key_x13, self.pub_key_x14, self.pub_key_x15,
         self.pub_key_x16, self.pub_key_x17, self.pub_key_x18, self.pub_key_x19,
         self.pub_key_x20, self.pub_key_x21, self.pub_key_x22, self.pub_key_x23,
         self.pub_key_x24, self.pub_key_x25, self.pub_key_x26, self.pub_key_x27,
         self.pub_key_x28, self.pub_key_x29, self.pub_key_x30, self.pub_key_x31]
    }

    pub fn get_pub_key_y(self) -> [u8; 32] {
        [self.pub_key_y0, self.pub_key_y1, self.pub_key_y2, self.pub_key_y3,
         self.pub_key_y4, self.pub_key_y5, self.pub_key_y6, self.pub_key_y7,
         self.pub_key_y8, self.pub_key_y9, self.pub_key_y10, self.pub_key_y11,
         self.pub_key_y12, self.pub_key_y13, self.pub_key_y14, self.pub_key_y15,
         self.pub_key_y16, self.pub_key_y17, self.pub_key_y18, self.pub_key_y19,
         self.pub_key_y20, self.pub_key_y21, self.pub_key_y22, self.pub_key_y23,
         self.pub_key_y24, self.pub_key_y25, self.pub_key_y26, self.pub_key_y27,
         self.pub_key_y28, self.pub_key_y29, self.pub_key_y30, self.pub_key_y31]
    }

    // Add ONE optimization - return Fields for ECDSA verification without breaking existing code
    pub fn get_pub_key_x_field(self) -> Field {
        let bytes = self.get_pub_key_x();
        Field::from_le_bytes(bytes)
    }

    pub fn get_pub_key_y_field(self) -> Field {
        let bytes = self.get_pub_key_y();
        Field::from_le_bytes(bytes)
    }
}

#[derive(Deserialize, Packable, Serialize)]
pub struct Guardian {
    pub address: Guardian_PK,
}

impl Guardian {
    pub fn new(address: [u8; 20], pub_key_x: [u8; 32], pub_key_y: [u8; 32]) -> Self {
        Guardian { address: Guardian_PK::new(address, pub_key_x, pub_key_y) }
    }
    
    pub fn default() -> Self {
        Guardian { address: Guardian_PK::default() }
    }

    pub fn get_address(g: Self) -> [u8; 20] {
        Guardian_PK::as_array(g.address)
    }

    pub fn get_pub_key_x(self) -> [u8; 32] {
        self.address.get_pub_key_x()
    }

    pub fn get_pub_key_y(self) -> [u8; 32] {
        self.address.get_pub_key_y()
    }

    // Add the same optimization here
    pub fn get_pub_key_x_field(self) -> Field {
        self.address.get_pub_key_x_field()
    }

    pub fn get_pub_key_y_field(self) -> Field {
        self.address.get_pub_key_y_field()
    }
}

#[derive(Deserialize, Serialize)]
pub struct Signature {
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
    pub guardian_index: u8
}

impl Signature {
    pub fn new(r: [u8; 32], s: [u8; 32], v: u8, guardian_index: u8) -> Self {
        Signature { r, s, v, guardian_index }
    }

    pub fn default() -> Self {
        Signature { r: [0; 32], s: [0; 32], v: 0, guardian_index: 0 }
    }

    pub fn from_bytes(bytes: [u8; 66]) -> Self {
        // Wormhole VAA signature format:
        // byte 0: guardian index
        // bytes 1-32: r (32 bytes) - BIG ENDIAN
        // bytes 33-64: s (32 bytes) - BIG ENDIAN  
        // byte 65: recovery id (v)
        
        let guardian_index = bytes[0];
        
        // Extract r (bytes 1-32) - keep as big endian
        let r = [bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7], bytes[8],
                 bytes[9], bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15], bytes[16],
                 bytes[17], bytes[18], bytes[19], bytes[20], bytes[21], bytes[22], bytes[23], bytes[24],
                 bytes[25], bytes[26], bytes[27], bytes[28], bytes[29], bytes[30], bytes[31], bytes[32]];
        
        // Extract s (bytes 33-64) - keep as big endian
        let s = [bytes[33], bytes[34], bytes[35], bytes[36], bytes[37], bytes[38], bytes[39], bytes[40],
                 bytes[41], bytes[42], bytes[43], bytes[44], bytes[45], bytes[46], bytes[47], bytes[48],
                 bytes[49], bytes[50], bytes[51], bytes[52], bytes[53], bytes[54], bytes[55], bytes[56],
                 bytes[57], bytes[58], bytes[59], bytes[60], bytes[61], bytes[62], bytes[63], bytes[64]];
        
        let v = bytes[65];
        
        Signature { r, s, v, guardian_index }
    }
}

#[derive(Deserialize, Serialize)]
pub struct Body {
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain_id: u16,
    pub emitter_address: AztecAddress,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: [u8; 1024],
}

impl Body {
    pub fn new(
        timestamp: u32,
        nonce: u32,
        emitter_chain_id: u16,
        emitter_address: AztecAddress,
        sequence: u64,
        consistency_level: u8,
        payload: [u8; 1024],
    ) -> Self {
        Body {
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload
        }
    }
    
    pub fn default() -> Self {
        Body {
            timestamp: 0,
            nonce: 0,
            emitter_chain_id: 0,
            emitter_address: AztecAddress::zero(),
            sequence: 0,
            consistency_level: 0,
            payload: [0; 1024],
        }
    }

    pub fn from_bytes(bytes: [u8; 1060]) -> Body {
        let timestamp: u32 = u32_from_u8s_le([bytes[0], bytes[1], bytes[2], bytes[3]]);
        let nonce: u32 = u32_from_u8s_le([bytes[4], bytes[5], bytes[6], bytes[7]]);
        let emitter_chain_id: u16 = u16_from_u8s_le([bytes[8], bytes[9]]);

        // Parse emitter bytes into aztec address
        let emitter_address_bytes: [u8; 32] = [
            bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15], bytes[16], bytes[17],
            bytes[18], bytes[19], bytes[20], bytes[21], bytes[22], bytes[23], bytes[24], bytes[25],
            bytes[26], bytes[27], bytes[28], bytes[29], bytes[30], bytes[31], bytes[32], bytes[33],
            bytes[34], bytes[35], bytes[36], bytes[37], bytes[38], bytes[39], bytes[40], bytes[41],
        ];

        let emitter_address_field: Field = Field::from_le_bytes(emitter_address_bytes);
        let emitter_address: AztecAddress = AztecAddress {
            inner: emitter_address_field,
        };
        
        let sequence: u64 = u64_from_u8s_le([bytes[42], bytes[43], bytes[44], bytes[45], bytes[46], bytes[47], bytes[48], bytes[49]]);
        let consistency_level: u8 = bytes[50];
        
        // Extract payload (1024 bytes starting from index 51)
        let mut payload = [0; 1024];
        for i in 0..1024 {
            payload[i] = bytes[51 + i];
        }
        
        Body {
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload
        }
    }
}

#[derive(Deserialize, Serialize)]
pub struct VAA {
    pub version: u8,
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain_id: u16,
    pub emitter_address: AztecAddress,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: [u8; 1024],
    pub guardian_set_index: u32,
    pub signatures: [Signature; 13],
    pub hash: [u8; 32], 
}

impl VAA {
    pub fn default() -> Self {
        VAA {
            version: 0,
            timestamp: 0,
            nonce: 0,
            emitter_chain_id: 0,
            emitter_address: AztecAddress::zero(),
            sequence: 0,
            consistency_level: 0,
            payload: [0; 1024],
            guardian_set_index: 0,
            signatures: [Signature::new([0;32], [0;32], 0, 0); 13],
            hash: [0; 32]
        }
    }

    pub fn new(
        version: u8,
        timestamp: u32,
        nonce: u32,
        emitter_chain_id: u16,
        emitter_address: AztecAddress,
        sequence: u64,
        consistency_level: u8,
        payload: [u8; 1024],
        guardian_set_index: u32,
        signatures: [Signature; 13],
        hash: [u8; 32]
    ) -> Self {
        VAA {
            version,
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload,
            guardian_set_index,
            signatures,
            hash
        }
    }

    pub fn get_guardian_set_index(vaa: Self) -> u32{
        vaa.guardian_set_index
    }

    pub fn get_timestamp(vaa: Self) -> u32 {
        vaa.timestamp
    }

    pub fn get_payload(vaa: Self) -> [u8; 1024] {
        vaa.payload
    }

    pub fn get_hash(vaa: Self) -> [u8; 32] {
        vaa.hash
    }

    pub fn get_emitter_chain_id(vaa: Self) -> u16 {
        vaa.emitter_chain_id
    }

    pub fn get_sequence(vaa: Self) -> u64 {
        vaa.sequence
    }

    pub fn get_consistency_level(vaa: Self) -> u8 {
        vaa.consistency_level
    }
}

// Helper functions
pub fn u32_from_u8s_le(bytes: [u8; 4]) -> u32 {
    let mut result = 0;
    for i in 0..4 {
        result |= (bytes[i as u32] as u32) << (i * 8);
    }
    result
}

pub fn u16_from_u8s_le(bytes: [u8; 2]) -> u16 {
    let mut result = 0;
    for i in 0..2 {
        result |= (bytes[i as u32] as u16) << (i * 8);
    }
    result
}

pub fn u64_from_u8s_le(bytes: [u8; 8]) -> u64 {
    let mut result = 0;
    for i in 0..8 {
        result |= (bytes[i as u32] as u64) << (i * 8);
    }
    result
}

// Governance structures
pub struct ContractUpgrade {
    pub module: [u8; 32],
    pub action: u8,
    pub chain: u16,
    pub new_contract: Field
}

pub struct GuardianSetUpgrade {
    pub module: [u8; 32],
    pub action: u8,
    pub chain: u16,
    pub new_guardian_set_index: u32
}

pub struct SetMessageFee {
    pub module: [u8; 32],
    pub action: u8,
    pub chain: u16,
    pub message_fee: Field
}

pub struct TransferFees {
    pub module: [u8; 32],
    pub action: u8,
    pub chain: u16,
    pub amount: Field,
    pub recipient: [u8; 32]
}

pub struct RecoverChainId {
    pub module: [u8; 32],
    pub action: u8,
    pub evm_chain_id: Field,
    pub new_chain_id: u16
}

#[derive(Deserialize, Packable, Serialize)]
pub struct EmitterRegistry {
    pub next_id: u64,
}

pub struct EmitterCapability {
    pub emitter: u64,
    pub sequence: u64,
}

impl EmitterRegistry {
    pub fn new() -> Self {
        EmitterRegistry { next_id: 0 }
    }

    pub fn get_next_id(&mut self) -> u64 {
        let id = self.next_id;
        self.next_id += 1;
        id
    }
}

impl EmitterCapability {
    pub fn new(registry: &mut EmitterRegistry) -> Self {
        let emitter = registry.get_next_id();
        registry.next_id += 1;
        EmitterCapability { emitter, sequence: 0 }
    }

    pub fn use_sequence(&mut self) -> u64 {
        let sequence = self.sequence;
        self.sequence += 1;
        sequence
    }
}

#[derive(Deserialize, Packable, Serialize)]
pub struct WormholeStorage {
    pub provider: Provider,
    pub guardian_set_index: u64,
    pub guardian_set_expiry: u64,
    pub message_fee: u128,
    pub emitter_registry: EmitterRegistry,
}

impl WormholeStorage {
    pub fn init(provider: Provider) -> Self {
        WormholeStorage {
            provider: provider,
            guardian_set_index: 0,
            guardian_set_expiry: 86400,
            message_fee: 1,
            emitter_registry: EmitterRegistry::new(),
        }
    }

    // Getters
    pub fn get_provider(state: Self) -> Provider {
        state.provider
    }

    pub fn get_message_fee(state: Self) -> u128 {
        state.message_fee
    }

    pub fn get_chain_id(state: Self) -> u16 {
        state.provider.chain_id
    }

    pub fn get_guardian_set_index(state: Self) -> u64 {
        state.guardian_set_index
    }

    pub fn get_guardian_set_expiry(state: Self) -> u64 {
        state.guardian_set_expiry
    }

    // Setters
    pub fn set_guardian_set_expiry(mut state: Self, guardian_set_expiry: u64) -> Self {
        state.guardian_set_expiry = guardian_set_expiry;
        state
    }

    pub fn set_guardian_set_index(mut state: Self, guardian_set_index: u64) -> Self {
        state.guardian_set_index = guardian_set_index;
        state
    }

    pub fn set_chain_id(mut state: Self, chain_id: u16) -> Self {
        state.provider.chain_id = chain_id;
        state
    }

    pub fn set_provider(mut state: Self, provider: Provider) -> Self {
        state.provider = provider;
        state
    }

    pub fn set_message_fee(mut state: Self, message_fee: u128) -> Self {
        state.message_fee = message_fee;
        state
    }
}