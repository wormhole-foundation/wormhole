mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::*;
use wormhole_io::Writeable;
use wormhole_raw_vaas::support::EncodedAmount;

use std::io;

pub fn new_sender_address(
    sender_authority: &Signer,
    cpi_program_id: Option<Pubkey>,
) -> Result<Pubkey> {
    match cpi_program_id {
        Some(program_id) => {
            let (expected_authority, _) = Pubkey::find_program_address(&[b"sender"], &program_id);
            require_eq!(sender_authority.key(), expected_authority);
            Ok(program_id)
        }
        None => Ok(sender_authority.key()),
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TransferWithMessage {
    pub norm_amount: EncodedAmount,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub sender: Pubkey,
    pub payload: Vec<u8>,
}

impl Writeable for TransferWithMessage {
    fn written_size(&self) -> usize {
        1 + 32 + 32 + 2 + 32 + 2 + 32 + self.payload.len()
    }

    fn write<W>(&self, writer: &mut W) -> io::Result<()>
    where
        Self: Sized,
        W: io::Write,
    {
        self.norm_amount.0.write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.redeemer.write(writer)?;
        self.redeemer_chain.write(writer)?;
        self.sender.to_bytes().write(writer)?;
        writer.write_all(&self.payload)?;
        Ok(())
    }
}
