use {
    crate::{
        accounts::{
            Account,
            Claim,
            ClaimSeeds,
        },
        instructions::Instruction,
        Config,
    },
    borsh::BorshSerialize,
    solana_program::{
        instruction::{
            AccountMeta,
            Instruction as SolanaInstruction,
        },
        pubkey::Pubkey,
    },
    wormhole::{
        Chain,
        WormholeError,
    },
};

#[derive(Debug, Eq, PartialEq, BorshSerialize)]
struct SetFeesData {}

pub fn set_fees(
    wormhole: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    emitter: Pubkey,
    sequence: u64,
) -> Result<SolanaInstruction, WormholeError> {
    let bridge = Config::key(&wormhole, ());
    let claim = Claim::key(
        &wormhole,
        ClaimSeeds {
            chain: Chain::Solana,
            emitter,
            sequence,
        },
    );

    Ok(SolanaInstruction {
        program_id: wormhole,
        accounts:   vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(bridge, false),
            AccountMeta::new_readonly(message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data:       (Instruction::SetFees, SetFeesData {}).try_to_vec().unwrap(),
    })
}
