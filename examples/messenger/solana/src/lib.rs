#![deny(unused_must_use)]

// A common serialization library used in the blockchain space, which we'll use to serialize our
// cross chain message payloads.
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

// Solana SDK imports to interact with the solana runtime.
use solana_program::account_info::{
    next_account_info,
    AccountInfo,
};
use solana_program::entrypoint::ProgramResult;
use solana_program::program::invoke_signed;
use solana_program::pubkey::Pubkey;
use solana_program::{
    entrypoint,
    msg,
};

// Import Solana Wormhole SDK.
use wormhole_sdk::{
    instructions::post_message,
    ConsistencyLevel,
    VAA,
};

// Our Payload, defined in a common library.
pub use messenger_common::Message;

pub mod instruction;

#[derive(BorshSerialize, BorshDeserialize, Clone)]
pub enum Instruction {
    /// This instruction is used to send a message to another chain by emitting it as a wormhole
    /// message targetting another users key.
    ///
    /// 0: Payer         [Signer]
    /// 1: Message       [Signer]
    /// 2: Worm Config   [PDA]
    /// 3: Worm Fee      [PDA]
    /// 4: Worm Sequence [PDA]
    /// 5: Emitter       [PDA, Signer]
    /// 6: Clock         [Program]         -- Needed for wormhole to take block times.
    /// 7: Rent          [Program]         -- Needed for wormhole fee calculation on the message account.
    /// 8: System        [Program]         -- Needed for wormhole to take fees.
    /// 9: Wormhole      [Program]         -- Needed for wormhole invoke_signed.
    SendMessage(Message, u32),

    /// This is the same as the above message, but the example handler is more low level.
    SendMessageRaw(Message, u32),

    /// This instruction receives a message by processing an incoming VAA containing a message
    /// intended for a receiver on Solana. Note that the simple existence of the VAA account is
    /// enough to verify it as the account is only created by the bridge if the guardians had
    /// successfull signed it.
    ///
    /// 0: VAA [PDA]
    RecvMessage,
}


entrypoint!(process_instruction);

/// The Solana entrypoint, here we deserialize our Borsh encoded Instruction and dispatch to our
/// program handlers.
pub fn process_instruction(id: &Pubkey, accs: &[AccountInfo], data: &[u8]) -> ProgramResult {
    match BorshDeserialize::try_from_slice(data).unwrap() {
        // Send Message Variants. Check the source of each to see various ways to invoke Wormhole.
        Instruction::SendMessage(msg, nonce)    => send_message(id, accs, msg, nonce),
        Instruction::SendMessageRaw(msg, nonce) => send_message_raw(id, accs, msg, nonce),

        // RecvMessage shows an example of safely processing a VAA.
        Instruction::RecvMessage         => recv_message(id, accs),
    }?;
    Ok(())
}

/// Send a Message from this chain to a user on a remote target chain.
///
/// This method is a reference example of emitting messages via Wormhole using the ergonomic API
/// methods. This is the easiest way to use Wormhole.
fn send_message(id: &Pubkey, accounts: &[AccountInfo], payload: Message, nonce: u32) -> ProgramResult {
    let iter    = &mut accounts.iter();
    let payer   = next_account_info(iter)?;
    let message = next_account_info(iter)?;

    // This helper method will take care of all of the following for you:
    //
    // - Derives a reasonable emitter PDA for your program.
    // - Pays the Bridge (Payer Key)
    // - Emits a Message
    wormhole_sdk::post_message(
        *id,
        *payer.key,
        *message.key,
        payload.try_to_vec()?,
        ConsistencyLevel::Finalized,
        None,
        accounts,
        nonce,
    )?;

    Ok(())
}

/// Send a Message from this chain to a user on a remote target chain.
///
/// This method is a reference example of emitting messages via Wormhole using the most low level
/// interface provided by the SDK. You must handle the emitter, payment, and invoking yourself.
fn send_message_raw(id: &Pubkey, accs: &[AccountInfo], payload: Message, nonce: u32) -> ProgramResult {
    let accounts      = &mut accs.iter();
    let payer         = next_account_info(accounts)?;
    let message       = next_account_info(accounts)?;
    let fee_collector = next_account_info(accounts)?;
    let config        = next_account_info(accounts)?;

    // Deserialize Bridge Config, used to figure out what the fee is so we can pay the bridge
    // programatically.
    let config = wormhole_sdk::read_config(config).unwrap();

    // Pay Fee to the Wormhole.
    invoke_signed(
        &solana_program::system_instruction::transfer(payer.key, fee_collector.key, config.fee),
        accs,
        &[],
    )?;

    // Create an Emitter to emit messages from, this helper method is producing the emitter from
    // the _current_ program's ID.
    let (emitter, mut seeds, bump) = wormhole_sdk::emitter(id);
    let bump = &[bump];
    seeds.push(bump);

    // Invoke the Wormhole post_message endpoint to create an on-chain message.
    invoke_signed(
        &post_message(
            wormhole_sdk::id(),
            *payer.key,
            emitter,
            *message.key,
            nonce,
            payload.try_to_vec()?,
            ConsistencyLevel::Finalized,
        )
        .unwrap(),
        accs,
        &[&seeds],
    )?;

    Ok(())
}


/// Receives a VAA containing a message from a foreign chain, and parses/verifies the VAA to
/// validate the message has been safely attested by the guardian set. Prints the message in
/// validator logs.
fn recv_message(id: &Pubkey, accs: &[AccountInfo]) -> ProgramResult {
    // We must verify the VAA is legitimately signed by the guardians. We do this by deriving the
    // expected PDA derived by the bridge, as long as we produce the same account we can trust the
    // contents of the VAA.
    let accounts = &mut accs.iter();
    let payer    = next_account_info(accounts)?;
    let vaa      = next_account_info(accounts)?;

    // If we want to avoid processing a message twice we need to track whether we have already
    // processed a VAA manually. There are several ways to do this in Solana but in this example
    // we will simply reprocess VAA's.
    let vaa = wormhole_sdk::read_vaa(vaa).unwrap();
    let msg = Message::try_from_slice(&vaa.payload)?;
    msg!("{}: {}", msg.nick, msg.text);

    Ok(())
}
