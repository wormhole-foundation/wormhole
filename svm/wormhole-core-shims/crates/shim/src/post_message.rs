//! Wormhole Post Message Shim program instruction and related types.

use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};
use wormhole_svm_definitions::{make_anchor_discriminator, EncodeFinality};

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PostMessageShimInstruction<'ix, F: EncodeFinality> {
    PostMessage(PostMessageData<'ix, F>),
}

impl<'ix, F: EncodeFinality> PostMessageShimInstruction<'ix, F> {
    pub const POST_MESSAGE_SELECTOR: [u8; 8] = make_anchor_discriminator(b"global:post_message");

    #[inline]
    pub fn to_vec(&self) -> Vec<u8> {
        match self {
            Self::PostMessage(data) => {
                let payload_len = data.payload.len();

                let mut out = Vec::with_capacity({
                    8 // selector
                    + 4 // nonce
                    + 4 // payload length
                    + payload_len // payload
                    + 1 // finality
                });
                out.extend_from_slice(&Self::POST_MESSAGE_SELECTOR);
                out.extend_from_slice(&data.nonce.to_le_bytes());
                out.push(data.finality.encode());
                out.extend_from_slice(&(payload_len as u32).to_le_bytes());
                out.extend_from_slice(data.payload);

                out
            }
        }
    }

    #[inline]
    pub fn deserialize(data: &'ix [u8]) -> Option<Self> {
        if data.len() < 8 {
            return None;
        }

        match data[..8].try_into().unwrap() {
            Self::POST_MESSAGE_SELECTOR => {
                PostMessageData::deserialize(&data[8..]).map(Self::PostMessage)
            }
            _ => None,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostMessageAccounts<'ix> {
    /// Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge
    /// program's post message instruction requires this account to be a signer.
    pub emitter: &'ix Pubkey,

    /// Payer will pay the rent for the Wormhole Core Bridge emitter sequence
    /// and message on the first post message call. Subsequent calls will not
    /// require more lamports for rent.
    pub payer: &'ix Pubkey,

    /// Wormhole Core Bridge program.
    pub wormhole_program_id: &'ix Pubkey,

    /// Program-derived accounts required for the post message instruction.
    ///
    /// NOTE: If `None` is used for any of these accounts,
    /// [PostMessage::instruction] will find these addresses. For on-chain
    /// applications, it is recommended to provide these accounts to save on
    /// CU costs (1,500 CU per bump iteration per account).
    pub derived: PostMessageDerivedAccounts<'ix>,
}

#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct PostMessageDerivedAccounts<'ix> {
    /// Wormhole Core Bridge config. The Wormhole Core Bridge program's post
    /// message instruction requires this account to be mutable.
    pub core_bridge_config: Option<&'ix Pubkey>,

    /// Wormhole Message. The Wormhole Core Bridge program's post message
    /// instruction requires this account to be a mutable signer.
    ///
    /// This program uses a PDA per emitter. Messages are already bottle-necked
    /// by emitter sequence and the Wormhole Core Bridge program enforces that
    /// emitter must be identical for reused accounts. While this could be
    /// managed by the integrator, it seems more effective to have this Shim
    /// program manage these accounts.
    pub message: Option<&'ix Pubkey>,

    /// Emitter's sequence account. Wormhole Core Bridge program's post message
    /// instruction requires this account to be mutable.
    pub sequence: Option<&'ix Pubkey>,

    /// Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post
    /// message instruction requires this account to be mutable.
    pub fee_collector: Option<&'ix Pubkey>,

    /// Wormhole Post Message Shim program's self-CPI authority.
    pub event_authority: Option<&'ix Pubkey>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PostMessageData<'ix, F: EncodeFinality> {
    /// Arbitrary message identifier specified by the message sender.
    nonce: u32,

    /// Finality of the message (which is when the Wormhole guardians will
    /// attest to this message's observation).
    finality: F,

    /// Message payload.
    payload: &'ix [u8],
}

impl<'ix, F: EncodeFinality> PostMessageData<'ix, F> {
    pub const MINIMUM_SIZE: usize = {
        4 // nonce
        + 1 // finality
        + 4 // payload length
    };

    #[inline]
    pub fn nonce(&self) -> u32 {
        self.nonce
    }

    #[inline]
    pub fn finality(&self) -> F {
        self.finality
    }

    #[inline]
    pub fn payload(&self) -> &'ix [u8] {
        self.payload
    }

    /// Construct new post message data. This method returns `None` if the
    /// payload length exceeds the maximum allowed size of [u32::MAX].
    pub fn new(nonce: u32, finality: F, payload: &'ix [u8]) -> Option<Self> {
        if payload.len() > u32::MAX as usize {
            None
        } else {
            Some(Self {
                nonce,
                finality,
                payload,
            })
        }
    }

    #[inline(always)]
    fn deserialize(data: &'ix [u8]) -> Option<Self> {
        if data.len() < Self::MINIMUM_SIZE {
            return None;
        }

        let nonce = u32::from_le_bytes(data[..4].try_into().unwrap());

        // NOTE: There may be different finality requirements among SVM
        // networks. This logic will have to change if that is the case.
        let finality = EncodeFinality::decode(data[4])?;

        let payload_len = u32::from_le_bytes(data[5..9].try_into().unwrap()) as usize;

        // This operation is unlikely to overflow (beware 32-bit architectures).
        // But it does not cost much to be paranoid.
        let total_len = Self::MINIMUM_SIZE.checked_add(payload_len)?;

        if data.len() < total_len {
            return None;
        }

        let payload = &data[9..(9 + payload_len)];

        // NOTE: We do not care about trailing bytes.

        Some(Self {
            nonce,
            payload,
            finality,
        })
    }
}

/// This instruction is intended to be a significantly cheaper alternative to
/// the post message instruction on Wormhole Core Bridge program. It achieves
/// this by reusing the message account (per emitter) via the post message
/// unreliable instruction and emitting data via self-CPI (Anchor event)
/// for the guardian to observe. This instruction data contains information
/// previously found only in the resulting message account.
///
/// Because this instruction passes through the emitter and calls the post
/// message unreliable instruction on the Wormhole Core Bridge, it can be used
/// without disruption.
///
/// NOTE: In the initial message publication for a new emitter, this will
/// require one additional CPI call depth when compared to using the Wormhole
/// Core Bridge directly. If this initial call depth is an issue, emit an empty
/// message on initialization (or migration) in order to instantiate the
/// message account. Posting a message will result in a VAA from your emitter,
/// so be careful to avoid any issues that may result from this first message.
///
/// Call depth of direct case:
/// 1. post message (Wormhole Post Message Shim)
/// 2. multiple CPI
///     - post message unreliable (Wormhole Core Bridge)
///     - Anchor event of `MesssageEvent` (Wormhole Post Message Shim)
///
/// Call depth of integrator case:
/// 1. integrator instruction
/// 2. CPI post message (Wormhole Post Message Shim)
/// 3. multiple CPI
///    - post message unreliable (Wormhole Core Bridge)
///    - Anchor event of `MesssageEvent` (Wormhole Post Message Shim)
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostMessage<'ix, F: EncodeFinality> {
    pub program_id: &'ix Pubkey,
    pub accounts: PostMessageAccounts<'ix>,
    pub data: PostMessageData<'ix, F>,
}

impl<F: EncodeFinality> PostMessage<'_, F> {
    /// Generate SVM instruction.
    #[inline]
    pub fn instruction(&self) -> Instruction {
        let program_id = self.program_id;

        let PostMessageAccounts {
            emitter,
            payer,
            wormhole_program_id,
            derived:
                PostMessageDerivedAccounts {
                    core_bridge_config,
                    message,
                    sequence,
                    fee_collector,
                    event_authority,
                },
        } = self.accounts;

        let core_bridge_config = core_bridge_config.copied().unwrap_or_else(|| {
            wormhole_svm_definitions::find_core_bridge_config_address(wormhole_program_id).0
        });
        let message = message.copied().unwrap_or_else(|| {
            wormhole_svm_definitions::find_shim_message_address(emitter, program_id).0
        });
        let sequence = sequence.copied().unwrap_or_else(|| {
            wormhole_svm_definitions::find_emitter_sequence_address(emitter, wormhole_program_id).0
        });
        let fee_collector = fee_collector.copied().unwrap_or_else(|| {
            wormhole_svm_definitions::find_fee_collector_address(wormhole_program_id).0
        });
        let event_authority = event_authority.copied().unwrap_or_else(|| {
            wormhole_svm_definitions::find_event_authority_address(program_id).0
        });

        Instruction {
            program_id: *program_id,
            accounts: vec![
                AccountMeta::new(core_bridge_config, false),
                AccountMeta::new(message, false),
                AccountMeta::new_readonly(*emitter, true),
                AccountMeta::new(sequence, false),
                AccountMeta::new(*payer, true),
                AccountMeta::new(fee_collector, false),
                AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
                AccountMeta::new_readonly(*wormhole_program_id, false),
                AccountMeta::new_readonly(event_authority, false),
                AccountMeta::new_readonly(*program_id, false),
            ],
            data: PostMessageShimInstruction::PostMessage(self.data).to_vec(),
        }
    }
}
