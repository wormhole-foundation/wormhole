use std::{
    io,
    ops::{Deref, DerefMut},
};

use crate::{
    error::CoreBridgeError,
    message::{WormDecode, WormEncode},
    state::{BridgeProgramData, PostedVaaV1},
};
use anchor_lang::prelude::*;

const GOVERNANCE_CHAIN: u16 = 1;
const GOVERNANCE_EMITTER: [u8; 32] = [
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4,
];

/// Governance module (A.K.A. "Core").
const GOVERNANCE_MODULE: [u8; 32] = [
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x43, 0x6f, 0x72, 0x65,
];

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GovernanceModule {
    label: [u8; 32],
}

impl From<[u8; 32]> for GovernanceModule {
    fn from(label: [u8; 32]) -> Self {
        GovernanceModule { label }
    }
}

impl WormDecode for GovernanceModule {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let mut label = [0; 32];
        reader.read_exact(&mut label)?;
        Ok(GovernanceModule { label })
    }
}

impl WormEncode for GovernanceModule {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        writer.write_all(&self.label)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GovernanceAction {
    value: u8,
}

impl From<u8> for GovernanceAction {
    fn from(value: u8) -> Self {
        GovernanceAction { value }
    }
}

impl WormDecode for GovernanceAction {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let value = u8::decode_reader(reader)?;
        Ok(Self { value })
    }
}

impl WormEncode for GovernanceAction {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.value.encode(writer)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TargetChain {
    Global,
    Solana,
}

impl WormDecode for TargetChain {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        match u16::decode_reader(reader)? {
            0 => Ok(TargetChain::Global),
            1 => Ok(TargetChain::Solana),
            _ => Err(io::ErrorKind::InvalidData.into()),
        }
    }
}

impl WormEncode for TargetChain {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        let chain = match self {
            TargetChain::Global => 0,
            TargetChain::Solana => 1,
        };
        u16::encode(&chain, writer)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GovernanceHeader {
    pub module: GovernanceModule,
    pub action: GovernanceAction,
    pub target: TargetChain,
}

impl WormDecode for GovernanceHeader {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let module = GovernanceModule::decode_reader(reader)?;
        let action = GovernanceAction::decode_reader(reader)?;
        let target = TargetChain::decode_reader(reader)?;
        Ok(GovernanceHeader {
            module,
            action,
            target,
        })
    }
}

impl WormEncode for GovernanceHeader {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.module.encode(writer)?;
        self.action.encode(writer)?;
        self.target.encode(writer)
    }
}

#[derive(Debug, Clone)]
pub struct GovernanceMessage<D: WormDecode + WormEncode> {
    pub header: GovernanceHeader,
    pub decree: D,
}

impl<D: WormDecode + WormEncode> Deref for GovernanceMessage<D> {
    type Target = GovernanceHeader;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}

impl<D: WormDecode + WormEncode> DerefMut for GovernanceMessage<D> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.header
    }
}

impl<D: WormDecode + WormEncode> WormDecode for GovernanceMessage<D> {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let header = GovernanceHeader::decode_reader(reader)?;
        let decree = D::decode_reader(reader)?;
        Ok(GovernanceMessage { header, decree })
    }
}

impl<D: WormDecode + WormEncode> WormEncode for GovernanceMessage<D> {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.header.encode(writer)?;
        self.decree.encode(writer)
    }
}

// Below is governance implementation for Core Bridge.
pub type PostedGovernanceVaaV1<D> = PostedVaaV1<GovernanceMessage<D>>;

pub(crate) fn require_valid_governance_posted_vaa<'ctx, D>(
    vaa: &'ctx Account<'_, PostedGovernanceVaaV1<D>>,
    bridge: &'ctx BridgeProgramData,
) -> Result<&'ctx GovernanceMessage<D>>
where
    D: Clone + WormDecode + WormEncode,
{
    // For the Core Bridge, we require that the current guardian set is used to sign this VAA.
    require!(
        bridge.guardian_set_index == vaa.guardian_set_index,
        CoreBridgeError::LatestGuardianSetRequired
    );

    require!(
        vaa.emitter_chain == GOVERNANCE_CHAIN.into()
            && vaa.emitter_address == GOVERNANCE_EMITTER.into(),
        CoreBridgeError::InvalidGovernanceEmitter
    );

    Ok(&vaa.payload)
}
