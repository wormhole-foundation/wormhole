use crate::{
    accounts::PostedVAADerivationData,
    api::{
        post_vaa::PostVAAData,
        ForeignAddress,
    },
    error::Error::{
        InvalidGovernanceAction,
        InvalidGovernanceChain,
        InvalidGovernanceModule,
    },
    PostedVAAData,
    Result,
    OUR_CHAIN_ID,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
};
use solitaire::{
    Context,
    Data,
    Peel,
    SolitaireError,
    *,
};
use std::{
    io::{
        Cursor,
        Read,
        Write,
    },
    ops::Deref,
};

pub trait SerializePayload: Sized {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), SolitaireError>;

    fn try_to_vec(&self) -> std::result::Result<Vec<u8>, SolitaireError> {
        let mut result = Vec::with_capacity(256);
        self.serialize(&mut result)?;
        Ok(result)
    }
}

pub trait DeserializePayload: Sized {
    fn deserialize(buf: &mut &[u8]) -> std::result::Result<Self, SolitaireError>;
}

pub trait SerializeGovernancePayload: SerializePayload {
    const MODULE: &'static str;
    const ACTION: u8;

    fn try_to_vec(&self) -> std::result::Result<Vec<u8>, SolitaireError> {
        let mut result = Vec::with_capacity(256);
        self.write_governance_header(&mut result)?;
        self.serialize(&mut result)?;
        Ok(result)
    }

    fn write_governance_header<W: Write>(
        &self,
        c: &mut W,
    ) -> std::result::Result<(), SolitaireError> {
        use byteorder::WriteBytesExt;
        let module = format!("{:\0>32}", Self::MODULE);
        let module = module.as_bytes();
        c.write_all(module)?;
        c.write_u8(Self::ACTION)?;
        c.write_u16::<BigEndian>(OUR_CHAIN_ID)?;
        Ok(())
    }
}

pub trait DeserializeGovernancePayload: DeserializePayload + SerializeGovernancePayload {
    fn check_governance_header(
        c: &mut Cursor<&mut &[u8]>,
    ) -> std::result::Result<(), SolitaireError> {
        let mut module = [0u8; 32];
        c.read_exact(&mut module)?;
        if module != format!("{:\0>32}", Self::MODULE).as_bytes() {
            return Err(InvalidGovernanceModule.into());
        }

        let action = c.read_u8()?;
        if action != Self::ACTION {
            return Err(InvalidGovernanceAction.into());
        }

        let chain = c.read_u16::<BigEndian>()?;
        if chain != OUR_CHAIN_ID && chain != 0 {
            return Err(InvalidGovernanceChain.into());
        }

        Ok(())
    }
}

pub struct PayloadMessage<'b, T: DeserializePayload>(
    Data<'b, PostedVAAData, { AccountState::Initialized }>,
    T,
);

impl<'a, 'b: 'a, T: DeserializePayload> Peel<'a, 'b> for PayloadMessage<'b, T> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self>
    where
        Self: Sized,
    {
        // Deserialize wrapped payload
        let data: Data<'b, PostedVAAData, { AccountState::Initialized }> = Data::peel(ctx)?;

        // Verify PDA derivation to ensure this account was created by the bridge
        // program via post_vaa. Without this check, an attacker could pass any
        // account with valid PostedVAAData layout but not actually derived from
        // guardian-verified VAA processing, bypassing signature verification.
        let body_hash: [u8; 32] = {
            let mut body_vec = Cursor::new(Vec::new());
            body_vec.write(&data.vaa_time.to_be_bytes())?;
            body_vec.write(&data.nonce.to_be_bytes())?;
            body_vec.write(&data.emitter_chain.to_be_bytes())?;
            body_vec.write(&data.emitter_address)?;
            body_vec.write(&data.sequence.to_be_bytes())?;
            body_vec.write(&[data.consistency_level])?;
            body_vec.write(&data.payload)?;
            let body = body_vec.into_inner();
            use sha3::Digest;
            let mut h = sha3::Keccak256::default();
            h.update(body.as_slice());
            h.finalize().into()
        };
        let msg_derivation = PostedVAADerivationData {
            payload_hash: body_hash.to_vec(),
        };
        data.verify_derivation(ctx.this, &msg_derivation)?;

        let payload = DeserializePayload::deserialize(&mut &data.payload[..])?;
        Ok(PayloadMessage(data, payload))
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        Data::persist(&self.0, program_id)
    }
}

impl<'b, T: DeserializePayload> Deref for PayloadMessage<'b, T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<'b, T: DeserializePayload> PayloadMessage<'b, T> {
    pub fn meta(&self) -> &PostedVAAData {
        &self.0
    }

    pub fn info(&self) -> AccountInfo<'b> {
        self.0.info().clone()
    }
}

pub struct SignatureItem {
    pub signature: Vec<u8>,
    pub key: [u8; 20],
    pub index: u8,
}

#[derive(Serialize, Deserialize, Default, Clone)]
pub struct VAASignature {
    pub signature: Vec<u8>,
    pub guardian_index: u8,
}

#[derive(Serialize, Deserialize, Default, Clone)]
pub struct VAA {
    // Header part
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<VAASignature>,
    // Body part
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u16,
    pub emitter_address: ForeignAddress,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: Vec<u8>,
}

impl VAA {
    pub const HEADER_LEN: usize = 6;
    pub const SIGNATURE_LEN: usize = 66;

    pub fn deserialize(data: &[u8]) -> std::result::Result<VAA, std::io::Error> {
        let mut rdr = Cursor::new(data);

        let version = rdr.read_u8()?;
        let guardian_set_index = rdr.read_u32::<BigEndian>()?;

        let len_sig = rdr.read_u8()?;
        let mut signatures: Vec<VAASignature> = Vec::with_capacity(len_sig as usize);
        for _i in 0..len_sig {
            let guardian_index = rdr.read_u8()?;
            let mut signature_data = [0u8; 65];
            rdr.read_exact(&mut signature_data)?;
            let signature = signature_data.to_vec();

            signatures.push(VAASignature {
                guardian_index,
                signature,
            });
        }

        let timestamp = rdr.read_u32::<BigEndian>()?;
        let nonce = rdr.read_u32::<BigEndian>()?;
        let emitter_chain = rdr.read_u16::<BigEndian>()?;

        let mut emitter_address = [0u8; 32];
        rdr.read_exact(&mut emitter_address)?;

        let sequence = rdr.read_u64::<BigEndian>()?;
        let consistency_level = rdr.read_u8()?;

        let mut payload = Vec::new();
        rdr.read_to_end(&mut payload)?;

        Ok(VAA {
            version,
            guardian_set_index,
            signatures,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            payload,
        })
    }
}

impl From<VAA> for PostVAAData {
    fn from(vaa: VAA) -> Self {
        PostVAAData {
            version: vaa.version,
            guardian_set_index: vaa.guardian_set_index,
            timestamp: vaa.timestamp,
            nonce: vaa.nonce,
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
            sequence: vaa.sequence,
            consistency_level: vaa.consistency_level,
            payload: vaa.payload,
        }
    }
}
