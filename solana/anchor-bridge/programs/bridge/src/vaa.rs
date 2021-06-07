use crate::{
    types::{
        ClaimData,
        PostedMessage,
    },
    Error::{
        InvalidGovernanceAction,
        InvalidGovernanceChain,
        InvalidGovernanceModule,
        VAAAlreadyExecuted,
    },
    Result,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
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
        if chain != 1 && chain != 0 {
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
        let data: Data<'b, PostedMessage, { AccountState::Initialized }> = Data::peel(ctx)?;
        // Deserialize wrapped payload
        let payload = DeserializePayload::deserialize(&mut &data.payload[..])?;

        Ok(PayloadMessage(data, payload))
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

data_wrapper!(Claim, ClaimData, AccountState::Uninitialized);

impl<'b, T: DeserializePayload> Seeded<&ClaimableVAA<'b, T>> for Claim<'b> {
    fn seeds(&self, _accs: &ClaimableVAA<'b, T>) -> Vec<Vec<u8>> {
        return vec![];
    }
}

#[derive(FromAccounts)]
pub struct ClaimableVAA<'b, T: DeserializePayload> {
    // Signed message for the transfer
    pub message: PayloadMessage<'b, T>, // TODO use bridge type here that does verifications

    // Claim account to prevent double spending
    pub claim: Claim<'b>,
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
        self.claim.verify_derivation(program_id, self)?;

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

        self.claim.create(self, ctx, payer, Exempt)?;
        self.claim.claimed = true;

        Ok(())
    }
}
