mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use ruint::aliases::U256;
use wormhole_io::Writeable;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Transfer {
    pub norm_amount: U256,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
    pub norm_relayer_fee: U256,
}

impl Transfer {
    const TYPE_ID: u8 = 1;
}

impl Writeable for Transfer {
    fn write<W>(&self, writer: &mut W) -> std::io::Result<()>
    where
        Self: Sized,
        W: std::io::Write,
    {
        Transfer::TYPE_ID.write(writer)?;
        self.norm_amount.write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.recipient.write(writer)?;
        self.recipient_chain.write(writer)?;
        self.norm_relayer_fee.write(writer)?;
        Ok(())
    }

    fn written_size(&self) -> usize {
        1 + 32 + 32 + 2 + 32 + 2 + 32
    }
}
