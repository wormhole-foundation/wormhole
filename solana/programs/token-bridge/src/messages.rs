//! Messages relevant to the Token Bridge across all networks. These messages are serialized and
//! then published via the Core Bridge program.

use anchor_lang::prelude::Pubkey;
use core_bridge_program::sdk::io::Writeable;
use ruint::aliases::U256;

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
        self.norm_amount.to_be_bytes::<32>().write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.recipient.write(writer)?;
        self.recipient_chain.write(writer)?;
        self.norm_relayer_fee.to_be_bytes::<32>().write(writer)?;
        Ok(())
    }

    fn written_size(&self) -> usize {
        1 + 32 + 32 + 2 + 32 + 2 + 32
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Attestation {
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub decimals: u8,
    pub symbol: [u8; 32],
    pub name: [u8; 32],
}

impl Attestation {
    const TYPE_ID: u8 = 2;
}

impl Writeable for Attestation {
    fn written_size(&self) -> usize {
        1 + 32 + 2 + 1 + 32 + 32
    }

    fn write<W>(&self, writer: &mut W) -> std::io::Result<()>
    where
        Self: Sized,
        W: std::io::Write,
    {
        Attestation::TYPE_ID.write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.decimals.write(writer)?;
        self.symbol.write(writer)?;
        self.name.write(writer)?;
        Ok(())
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TransferWithMessage {
    pub norm_amount: U256,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub sender: Pubkey,
    pub payload: Vec<u8>,
}

impl TransferWithMessage {
    const TYPE_ID: u8 = 3;
}

impl Writeable for TransferWithMessage {
    fn written_size(&self) -> usize {
        1 + 32 + 32 + 2 + 32 + 2 + 32 + self.payload.len()
    }

    fn write<W>(&self, writer: &mut W) -> std::io::Result<()>
    where
        Self: Sized,
        W: std::io::Write,
    {
        TransferWithMessage::TYPE_ID.write(writer)?;
        self.norm_amount.to_be_bytes::<32>().write(writer)?;
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.redeemer.write(writer)?;
        self.redeemer_chain.write(writer)?;
        self.sender.to_bytes().write(writer)?;
        writer.write_all(&self.payload)?;
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use crate::legacy::string_to_fixed32;
    use hex_literal::hex;
    use wormhole_raw_vaas::token_bridge;

    use super::*;

    #[test]
    fn transfer() {
        let transfer = Transfer {
            norm_amount: U256::from(69420u64),
            token_address: hex!("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
            token_chain: 2,
            recipient: hex!("d00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00d"),
            recipient_chain: 1,
            norm_relayer_fee: U256::from(42069u64),
        };

        let mut bytes = Vec::with_capacity(transfer.written_size());
        transfer.write(&mut bytes).unwrap();

        let msg = token_bridge::TokenBridgeMessage::parse(&bytes).unwrap();
        let parsed = msg.transfer().unwrap();

        let expected = Transfer {
            norm_amount: U256::from_be_bytes(parsed.amount()),
            token_address: parsed.token_address(),
            token_chain: parsed.token_chain(),
            recipient: parsed.recipient(),
            recipient_chain: parsed.recipient_chain(),
            norm_relayer_fee: U256::from_be_bytes(parsed.relayer_fee()),
        };
        assert_eq!(transfer, expected);
    }

    #[test]
    fn attestation() {
        let symbol = "WETH".to_string();
        let name = "Wrapped Ether".to_string();

        let attestation = Attestation {
            token_address: hex!("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
            token_chain: 2,
            decimals: 18,
            symbol: string_to_fixed32(&symbol),
            name: string_to_fixed32(&name),
        };

        let mut bytes = Vec::with_capacity(attestation.written_size());
        attestation.write(&mut bytes).unwrap();

        let msg = token_bridge::TokenBridgeMessage::parse(&bytes).unwrap();
        let parsed = msg.attestation().unwrap();

        let expected = Attestation {
            token_address: parsed.token_address(),
            token_chain: parsed.token_chain(),
            decimals: parsed.decimals(),
            symbol: string_to_fixed32(&parsed.symbol().to_string()),
            name: string_to_fixed32(&parsed.name().to_string()),
        };
        assert_eq!(attestation, expected);
    }

    #[test]
    fn transfer_with_message() {
        let transfer = TransferWithMessage {
            norm_amount: U256::from(69420u64),
            token_address: hex!("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
            token_chain: 2,
            redeemer: hex!("d00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00dd00d"),
            redeemer_chain: 1,
            sender: solana_program::sysvar::clock::id(),
            payload: b"All your base are belong to us.".to_vec(),
        };

        let mut bytes = Vec::with_capacity(transfer.written_size());
        transfer.write(&mut bytes).unwrap();

        let msg = token_bridge::TokenBridgeMessage::parse(&bytes).unwrap();
        let parsed = msg.transfer_with_message().unwrap();

        let expected = TransferWithMessage {
            norm_amount: U256::from_be_bytes(parsed.amount()),
            token_address: parsed.token_address(),
            token_chain: parsed.token_chain(),
            redeemer: parsed.redeemer(),
            redeemer_chain: parsed.redeemer_chain(),
            sender: parsed.sender().into(),
            payload: parsed.payload().to_vec(),
        };
        assert_eq!(transfer, expected);
    }
}
