use {
    crate::{
        accounts::Account,
        instructions::Instruction,
        Config,
        FeeCollector,
        GuardianSet,
    },
    borsh::BorshSerialize,
    solana_program::{
        instruction::{
            AccountMeta,
            Instruction as SolanaInstruction,
        },
        pubkey::Pubkey,
        sysvar,
    },
    wormhole::WormholeError,
};

#[derive(BorshSerialize, Default)]
pub struct InitializeData {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_expiration_time: u32,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee: u64,

    /// Initial Guardian Set
    pub initial_guardians: Vec<[u8; 20]>,
}

pub fn initialize(
    wormhole: Pubkey,
    payer: Pubkey,
    fee: u64,
    guardian_set_expiration_time: u32,
    initial_guardians: &[[u8; 20]],
) -> Result<SolanaInstruction, WormholeError> {
    let bridge = Config::key(&wormhole, ());
    let guardian_set = GuardianSet::key(&wormhole, 0);
    let fee_collector = FeeCollector::key(&wormhole, ());

    Ok(SolanaInstruction {
        program_id: wormhole,
        accounts:   vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(guardian_set, false),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data:       (
            Instruction::Initialize,
            InitializeData {
                initial_guardians: initial_guardians.to_vec(),
                fee,
                guardian_set_expiration_time,
            },
        )
            .try_to_vec()?,
    })
}
