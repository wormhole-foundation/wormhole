use crate::{
    accounts::{
        AuthoritySigner,
        CustodySigner,
        FromCustodyTokenAccount,
        FromCustodyTokenAccountDerivationData,
        MigrationPool,
        MigrationPoolDerivationData,
        ShareMint,
        ShareMintDerivationData,
        ToCustodyTokenAccount,
        ToCustodyTokenAccountDerivationData,
    },
    api::{
        add_liquidity::AddLiquidityData,
        claim_shares::ClaimSharesData,
        create_pool::CreatePoolData,
        migrate_tokens::MigrateTokensData,
        remove_liquidity::RemoveLiquidityData,
    },
};
use borsh::BorshSerialize;
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

pub fn add_liquidity(
    program_id: Pubkey,
    from_mint: Pubkey,
    to_mint: Pubkey,
    liquidity_token_account: Pubkey,
    lp_share_token_account: Pubkey,
    amount: u64,
) -> solitaire::Result<Instruction> {
    let pool = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint,
            to: to_mint,
        },
        &program_id,
    );
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(pool, false),
            AccountMeta::new_readonly(from_mint, false),
            AccountMeta::new_readonly(to_mint, false),
            AccountMeta::new(
                ToCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &ToCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                ShareMint::<'_, { AccountState::Uninitialized }>::key(
                    &ShareMintDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(liquidity_token_account, false),
            AccountMeta::new(lp_share_token_account, false),
            AccountMeta::new_readonly(CustodySigner::key(None, &program_id), false),
            AccountMeta::new_readonly(AuthoritySigner::key(None, &program_id), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::AddLiquidity,
            AddLiquidityData { amount },
        )
            .try_to_vec()?,
    })
}

pub fn remove_liquidity(
    program_id: Pubkey,
    from_mint: Pubkey,
    to_mint: Pubkey,
    liquidity_token_account: Pubkey,
    lp_share_token_account: Pubkey,
    amount: u64,
) -> solitaire::Result<Instruction> {
    let pool = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint,
            to: to_mint,
        },
        &program_id,
    );
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(pool, false),
            AccountMeta::new_readonly(from_mint, false),
            AccountMeta::new_readonly(to_mint, false),
            AccountMeta::new(
                ToCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &ToCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                ShareMint::<'_, { AccountState::Uninitialized }>::key(
                    &ShareMintDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(liquidity_token_account, false),
            AccountMeta::new(lp_share_token_account, false),
            AccountMeta::new_readonly(CustodySigner::key(None, &program_id), false),
            AccountMeta::new_readonly(AuthoritySigner::key(None, &program_id), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::RemoveLiquidity,
            RemoveLiquidityData { amount },
        )
            .try_to_vec()?,
    })
}

pub fn claim_shares(
    program_id: Pubkey,
    from_mint: Pubkey,
    to_mint: Pubkey,
    output_token_account: Pubkey,
    lp_share_token_account: Pubkey,
    amount: u64,
) -> solitaire::Result<Instruction> {
    let pool = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint,
            to: to_mint,
        },
        &program_id,
    );
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(pool, false),
            AccountMeta::new(
                FromCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &FromCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                ShareMint::<'_, { AccountState::Uninitialized }>::key(
                    &ShareMintDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(output_token_account, false),
            AccountMeta::new(lp_share_token_account, false),
            AccountMeta::new_readonly(CustodySigner::key(None, &program_id), false),
            AccountMeta::new_readonly(AuthoritySigner::key(None, &program_id), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::ClaimShares,
            ClaimSharesData { amount },
        )
            .try_to_vec()?,
    })
}

pub fn create_pool(
    program_id: Pubkey,
    payer: Pubkey,
    from_mint: Pubkey,
    to_mint: Pubkey,
) -> solitaire::Result<Instruction> {
    let pool = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint,
            to: to_mint,
        },
        &program_id,
    );
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(pool, false),
            AccountMeta::new_readonly(from_mint, false),
            AccountMeta::new_readonly(to_mint, false),
            AccountMeta::new(
                FromCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &FromCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                ToCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &ToCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                ShareMint::<'_, { AccountState::Uninitialized }>::key(
                    &ShareMintDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new_readonly(CustodySigner::key(None, &program_id), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::CreatePool,
            CreatePoolData {},
        )
            .try_to_vec()?,
    })
}

pub fn migrate_tokens(
    program_id: Pubkey,
    from_mint: Pubkey,
    to_mint: Pubkey,
    input_token_account: Pubkey,
    output_token_account: Pubkey,
    amount: u64,
) -> solitaire::Result<Instruction> {
    let pool = MigrationPool::<'_, { AccountState::Initialized }>::key(
        &MigrationPoolDerivationData {
            from: from_mint,
            to: to_mint,
        },
        &program_id,
    );
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(pool, false),
            AccountMeta::new_readonly(from_mint, false),
            AccountMeta::new_readonly(to_mint, false),
            AccountMeta::new(
                ToCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &ToCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(
                FromCustodyTokenAccount::<'_, { AccountState::Uninitialized }>::key(
                    &FromCustodyTokenAccountDerivationData { pool },
                    &program_id,
                ),
                false,
            ),
            AccountMeta::new(input_token_account, false),
            AccountMeta::new(output_token_account, false),
            AccountMeta::new_readonly(CustodySigner::key(None, &program_id), false),
            AccountMeta::new_readonly(AuthoritySigner::key(None, &program_id), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::MigrateTokens,
            MigrateTokensData { amount },
        )
            .try_to_vec()?,
    })
}
