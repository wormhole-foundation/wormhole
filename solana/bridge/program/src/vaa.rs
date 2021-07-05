use crate::{
    accounts::{
        Claim,
        ClaimDerivationData,
    },
    error::Error::{
        InvalidGovernanceAction,
        InvalidGovernanceChain,
        InvalidGovernanceModule,
        VAAAlreadyExecuted,
    },
    types::PostedMessage,
    Result,
    CHAIN_ID_SOLANA,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
};
use solana_program::{
    instruction::AccountMeta,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::Seeded,
    trace,
    Context,
    CreationLamports::Exempt,
    Data,
    ExecutionContext,
    InstructionContext,
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

pub trait DeserializeGovernancePayload {
    const MODULE: &'static str;
    const ACTION: u8;

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
    Data<'b, PostedMessage, { AccountState::Initialized }>,
    T,
);

impl<'a, 'b: 'a, 'c, T: DeserializePayload> Peel<'a, 'b, 'c> for PayloadMessage<'b, T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized,
    {
        // Deserialize wrapped payload
        let data: Data<'b, PostedMessage, { AccountState::Initialized }> = Data::peel(ctx)?;
        let payload = DeserializePayload::deserialize(&mut &data.payload[..])?;
        Ok(PayloadMessage(data, payload))
    }

    fn deps() -> Vec<Pubkey> {
        Data::<'b, PostedMessage, { AccountState::Initialized }>::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        Data::persist(&self.0, program_id)
    }

    fn to_partial_cpi_metas(infos: &'c mut std::slice::Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        Data::<'b, PostedMessage, {AccountState::Initialized}>::to_partial_cpi_metas(infos)
    }
}

impl<'b, T: DeserializePayload> Deref for PayloadMessage<'b, T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<'b, T: DeserializePayload> PayloadMessage<'b, T> {
    pub fn meta(&self) -> &PostedMessage {
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

impl<'b, T: DeserializePayload> InstructionContext<'b> for ClaimableVAA<'b, T> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        // Do the Posted Message verification

        // Verify that the claim account is derived correctly
        trace!("Seq: {}", self.message.meta().sequence);
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
