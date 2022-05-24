use bridge::types::ConsistencyLevel;
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
};

#[allow(clippy::too_many_arguments)]
pub fn post_message(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: ConsistencyLevel,
) -> solitaire::Result<Instruction> {
    let ix = bridge::instructions::post_message(
        bridge_id, payer, emitter, message, nonce, payload, commitment,
    )?;
    let mut accounts = ix.accounts;
    accounts.insert(7, AccountMeta::new_readonly(bridge_id, false));
    let mut data = ix.data;
    data[0] = 0;

    Ok(Instruction {
        program_id,
        accounts,
        data,
    })
}
