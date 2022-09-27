use {
    crate::{
        accounts::Account,
        instructions::Instruction,
        Config,
        GuardianSet,
        VAA,
    },
    borsh::BorshSerialize,
    byteorder::{
        BigEndian,
        WriteBytesExt,
    },
    sha3::Digest,
    solana_program::{
        instruction::{
            AccountMeta,
            Instruction as SolanaInstruction,
        },
        pubkey::Pubkey,
        system_program,
        sysvar,
    },
    std::io::{
        Cursor,
        Write,
    },
    wormhole::WormholeError,
};

#[derive(Debug, Eq, PartialEq, BorshSerialize)]
pub struct PostVAAData {
    // Header part
    pub version:            u8,
    pub guardian_set_index: u32,

    // Body part
    pub timestamp:         u32,
    pub nonce:             u32,
    pub emitter_chain:     u16,
    pub emitter_address:   [u8; 32],
    pub sequence:          u64,
    pub consistency_level: u8,
    pub payload:           Vec<u8>,
}

impl PostVAAData {
    pub fn hash(&self) -> [u8; 32] {
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(self.timestamp).unwrap();
            v.write_u32::<BigEndian>(self.nonce).unwrap();
            v.write_u16::<BigEndian>(self.emitter_chain).unwrap();
            v.write_all(&self.emitter_address).unwrap();
            v.write_u64::<BigEndian>(self.sequence).unwrap();
            v.write_u8(self.consistency_level).unwrap();
            v.write_all(&self.payload).unwrap();
            v.into_inner()
        };
        let mut h = sha3::Keccak256::default();
        h.write_all(body.as_slice()).unwrap();
        h.finalize().into()
    }
}

pub fn post_vaa(
    wormhole: Pubkey,
    payer: Pubkey,
    signature_set: Pubkey,
    post_vaa_data: PostVAAData,
) -> Result<SolanaInstruction, WormholeError> {
    let bridge = Config::key(&wormhole, ());
    let guardian_set = GuardianSet::key(&wormhole, post_vaa_data.guardian_set_index);
    let vaa = VAA::key(&wormhole, post_vaa_data.hash());

    Ok(SolanaInstruction {
        program_id: wormhole,
        accounts:   vec![
            AccountMeta::new_readonly(guardian_set, false),
            AccountMeta::new_readonly(bridge, false),
            AccountMeta::new_readonly(signature_set, false),
            AccountMeta::new(vaa, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(system_program::id(), false),
        ],
        data:       (Instruction::PostVAA, post_vaa_data).try_to_vec()?,
    })
}
