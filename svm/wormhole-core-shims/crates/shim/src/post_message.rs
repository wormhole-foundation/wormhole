use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};

pub const POST_MESSAGE_SELECTOR: [u8; 8] = trim_digest(
    sha2_const_stable::Sha256::new()
        .update(b"global:post_message")
        .finalize(),
);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostMessageData<'data> {
    /// Arbitrary message identifier specified by the message sender.
    pub nonce: u32,

    /// Message payload.
    pub payload: &'data [u8],

    /// Whether the message should be finalized. If true, the message will be
    /// observed by the Wormhole guardians at the Finalized commitment level
    /// (32 slots). Otherwise it will be observed at the Confirmed commitment
    /// level (1 slot).
    pub finality: Finality,
}

impl<'data> PostMessageData<'data> {
    #[inline]
    pub fn to_vec(&self) -> Vec<u8> {
        let mut data = Vec::with_capacity(
            8 // selector
            + 4 // nonce
            + 4 // payload length
            + self.payload.len() // payload
            + 1, // finality
        );
        data.extend_from_slice(&POST_MESSAGE_SELECTOR);
        data.extend_from_slice(&self.nonce.to_le_bytes());
        data.push(self.finality as u8);
        data.extend_from_slice(&(self.payload.len() as u32).to_le_bytes());
        data.extend_from_slice(self.payload);

        data
    }

    #[inline]
    pub fn deserialize(data: &'data [u8]) -> Option<Self> {
        const MINIMUM_SIZE: usize = {
            8 // selector
            + 4 // nonce
            + 4 // payload length
            + 1 //. finality
        };
        if data.len() < MINIMUM_SIZE || data[..8] != POST_MESSAGE_SELECTOR {
            return None;
        }

        let nonce = u32::from_le_bytes(data[8..12].try_into().unwrap());

        // NOTE: There may be different finality requirements among SVM
        // networks. This logic will have to change if that is the case.
        let finality = match data[12] {
            0 => Finality::Confirmed,
            1 => Finality::Finalized,
            _ => return None,
        };

        let payload_len = u32::from_le_bytes(data[13..17].try_into().unwrap()) as usize;

        if data.len() < MINIMUM_SIZE.saturating_add(payload_len) {
            return None;
        }

        let payload = &data[17..17 + payload_len];

        // NOTE: We do not care about trailing bytes.

        Some(Self {
            nonce,
            payload,
            finality,
        })
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[repr(u8)]
pub enum Finality {
    Confirmed,
    Finalized,
}

/// This instruction is intended to be a significantly cheaper alternative to
/// `post_message` on the core bridge. It achieves this by reusing the message
/// account, per emitter, via `post_message_unreliable` and emitting a CPI event
/// for the guardian to observe containing the information previously only found
/// in the resulting message account. Since this passes through the emitter and
/// calls `post_message_unreliable` on the core bridge, it can be used (or not
/// used) without disruption.
///
/// NOTE: In the initial message publication for a new emitter, this will
/// require one additional CPI call depth when compared to using the core bridge
/// directly. If that is an issue, simply emit an empty message on
/// initialization (or migration) in order to instantiate the account. This will
/// result in a VAA from your emitter, so be careful to avoid any issues that
/// may result in.
///
/// Direct case
/// shim `PostMessage` -> core `0x8`
///                    -> shim `MesssageEvent`
///
/// Integration case
/// Integrator Program -> shim `PostMessage` -> core `0x8`
///                                          -> shim `MesssageEvent`
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostMessage<'ix> {
    // Accounts.
    //
    /// Wormhole Core Bridge config. Wormhole Core Bridge program's post message
    /// instruction requires this account be mutable.
    pub core_bridge_config: &'ix Pubkey,

    /// Wormhole Message. Wormhole Core Bridge program's post message
    /// instruction requires this account to be a mutable signer.
    ///
    /// The Wormhole Post Message Shim program uses a PDA per emitter because
    /// these messages are already bottle-necked by sequence and the Wormhole
    /// Core Bridge program enforces that the emitter must be identical for
    /// reused accounts.
    ///
    /// While this could be managed by the integrator, it seems more effective
    /// to have the Wormhole Post Message Shim program manage these accounts.
    pub message: &'ix Pubkey,

    /// Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge
    /// program's post message instruction requires this account to be a signer.
    pub emitter: &'ix Pubkey,

    /// Emitter's sequence account. Wormhole Core Bridge program's post message
    /// instruction requires this account to be mutable.
    pub sequence: &'ix Pubkey,

    /// Payer will pay the Wormhole Core Bridge fee to post a message. The fee
    /// amount is encoded in the [core_bridge_config] account data.
    ///
    /// [core_bridge_config]: Self::core_bridge_config
    pub payer: &'ix Pubkey,

    /// Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post
    /// message instruction requires this account to be mutable.
    pub fee_collector: &'ix Pubkey,

    /// Wormhole Core Bridge program.
    pub core_bridge_program: &'ix Pubkey,

    /// Wormhole Post Message Shim program's self-CPI authority.
    pub event_authority: Option<&'ix Pubkey>,

    /// Instruction data.
    pub data: PostMessageData<'ix>,
}

impl<'data> PostMessage<'data> {
    /// Generate SVM instruction for given Wormhole Post Message Shim program.
    #[inline]
    pub fn instruction(&self, program_id: &Pubkey) -> Instruction {
        Instruction {
            program_id: *program_id,
            accounts: vec![
                AccountMeta::new(*self.core_bridge_config, false),
                AccountMeta::new(*self.message, false),
                AccountMeta::new_readonly(*self.emitter, true),
                AccountMeta::new(*self.sequence, false),
                AccountMeta::new(*self.payer, true),
                AccountMeta::new(*self.fee_collector, false),
                AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
                AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
                AccountMeta::new_readonly(*self.core_bridge_program, false),
                AccountMeta::new_readonly(
                    self.event_authority.copied().unwrap_or_else(|| {
                        wormhole_svm_definitions::find_event_authority_address(program_id).0
                    }),
                    false,
                ),
                AccountMeta::new_readonly(*program_id, false),
            ],
            data: self.data.to_vec(),
        }
    }
}

const fn trim_digest(digest: [u8; 32]) -> [u8; 8] {
    let mut trimmed = [0; 8];
    let mut i = 0;

    loop {
        if i >= 8 {
            break;
        }
        trimmed[i] = digest[i];
        i += 1;
    }

    trimmed
}
