use crate::{
    accounts::{
        Claim,
        ClaimDerivationData,
    },
    api::{
        post_vaa::PostVAAData,
        ForeignAddress,
    },
    error::Error::{
        InvalidGovernanceAction,
        InvalidGovernanceChain,
        InvalidGovernanceModule,
        VAAAlreadyExecuted,
    },
    types::PostedVAAData,
    Result,
    CHAIN_ID_SOLANA,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    trace,
    Context,
    CreationLamports::Exempt,
    Data,
    ExecutionContext,
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
        c.write(&module)?;
        c.write_u8(Self::ACTION)?;
        c.write_u16::<BigEndian>(CHAIN_ID_SOLANA)?;
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
        if chain != CHAIN_ID_SOLANA && chain != 0 {
            return Err(InvalidGovernanceChain.into());
        }

        Ok(())
    }
}

pub struct PayloadMessage<'b, T: DeserializePayload>(
    Data<'b, PostedVAAData, { AccountState::Initialized }>,
    T,
);

impl<'a, 'b: 'a, 'c, T: DeserializePayload> Peel<'a, 'b, 'c> for PayloadMessage<'b, T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized,
    {
        // Deserialize wrapped payload
        let data: Data<'b, PostedVAAData, { AccountState::Initialized }> = Data::peel(ctx)?;
        let payload = DeserializePayload::deserialize(&mut &data.payload[..])?;
        Ok(PayloadMessage(data, payload))
    }

    fn deps() -> Vec<Pubkey> {
        Data::<'b, PostedVAAData, { AccountState::Initialized }>::deps()
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
}

#[derive(FromAccounts)]
pub struct ClaimableVAA<'b, T: DeserializePayload> {
    // Signed message for the transfer
    pub message: PayloadMessage<'b, T>,

    // Claim account to prevent double spending
    pub claim: Mut<Claim<'b, { AccountState::Uninitialized }>>,
}

impl<'b, T: DeserializePayload> Deref for ClaimableVAA<'b, T> {
    type Target = PayloadMessage<'b, T>;
    fn deref(&self) -> &Self::Target {
        &self.message
    }
}

impl<'b, T: DeserializePayload> ClaimableVAA<'b, T> {
    pub fn verify(&self, program_id: &Pubkey) -> Result<()> {
        trace!("Seq: {}", self.message.meta().sequence);

        // Verify that the claim account is derived correctly
        self.claim.verify_derivation(
            program_id,
            &ClaimDerivationData {
                emitter_address: self.message.meta().emitter_address,
                emitter_chain: self.message.meta().emitter_chain,
                sequence: self.message.meta().sequence,
            },
        )?;

        Ok(())
    }
}

impl<'b, T: DeserializePayload> ClaimableVAA<'b, T> {
    pub fn is_claimed(&self) -> bool {
        self.claim.claimed
    }

    pub fn claim(&mut self, ctx: &ExecutionContext, payer: &Pubkey) -> Result<()> {
        if self.is_claimed() {
            return Err(VAAAlreadyExecuted.into());
        }

        self.claim.create(
            &ClaimDerivationData {
                emitter_address: self.message.meta().emitter_address,
                emitter_chain: self.message.meta().emitter_chain,
                sequence: self.message.meta().sequence,
            },
            ctx,
            payer,
            Exempt,
        )?;

        self.claim.claimed = true;

        Ok(())
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
        let mut v = VAA::default();

        v.version = rdr.read_u8()?;
        v.guardian_set_index = rdr.read_u32::<BigEndian>()?;

        let len_sig = rdr.read_u8()?;
        let mut sigs: Vec<VAASignature> = Vec::with_capacity(len_sig as usize);
        for _i in 0..len_sig {
            let mut sig = VAASignature::default();

            sig.guardian_index = rdr.read_u8()?;
            let mut signature_data = [0u8; 65];
            rdr.read_exact(&mut signature_data)?;
            sig.signature = signature_data.to_vec();

            sigs.push(sig);
        }
        v.signatures = sigs;

        v.timestamp = rdr.read_u32::<BigEndian>()?;
        v.nonce = rdr.read_u32::<BigEndian>()?;
        v.emitter_chain = rdr.read_u16::<BigEndian>()?;

        let mut emitter_address = [0u8; 32];
        rdr.read_exact(&mut emitter_address)?;
        v.emitter_address = emitter_address;

        v.sequence = rdr.read_u64::<BigEndian>()?;
        v.consistency_level = rdr.read_u8()?;

        rdr.read_to_end(&mut v.payload)?;

        Ok(v)
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
