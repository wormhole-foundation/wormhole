mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use wormhole_io::Writeable;
use wormhole_raw_vaas::support::EncodedAmount;

use std::io;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Transfer {
    pub norm_amount: EncodedAmount,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
    pub norm_relayer_fee: EncodedAmount,
}

impl Transfer {
    const TYPE_ID: u8 = 1;
}

impl Writeable for Transfer {
    fn write<W>(&self, writer: &mut W) -> io::Result<()>
    where
        Self: Sized,
        W: io::Write,
    {
        Transfer::TYPE_ID.write(writer)?;
        self.norm_amount.0.write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.recipient.write(writer)?;
        self.recipient_chain.write(writer)?;
        self.norm_relayer_fee.0.write(writer)?;
        Ok(())
    }

    fn written_size(&self) -> usize {
        1 + 32 + 32 + 2 + 32 + 2 + 32
    }
}
