use crate::{
    accounts::{
        AuthoritySigner,
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        EmitterAccount,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    api::{
        complete_transfer::{
            CompleteNativeData,
            CompleteWrappedData,
        },
        RegisterChainData,
        TransferNativeData,
        TransferWrappedData,
        UpgradeContractData,
    },
    messages::{
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    },
    CompleteWrappedMetaData,
};
use borsh::BorshSerialize;
use bridge::{
    accounts::{
        Bridge,
        Claim,
        ClaimDerivationData,
        FeeCollector,
        Sequence,
        SequenceDerivationData,
    },
    api::ForeignAddress,
    PostVAAData,
    CHAIN_ID_SOLANA,
};
use primitive_types::U256;
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

pub fn initialize(
    program_id: Pubkey,
    payer: Pubkey,
    bridge: Pubkey,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
        ],
        data: (crate::instruction::Instruction::Initialize, bridge).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_native(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    to_authority: Pubkey,
    mint: Pubkey,
    data: CompleteNativeData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );
    let custody_signer_key = CustodySigner::key(None, &program_id);
    let associated_addr =
        spl_associated_token_account::get_associated_token_address(&to_authority, &mint);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(associated_addr, false),
            AccountMeta::new_readonly(to_authority, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
            AccountMeta::new_readonly(spl_associated_token_account::id(), false),
        ],
        data: (crate::instruction::Instruction::CompleteNative, data).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadTransfer,
    to_authority: Pubkey,
    data: CompleteWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
            token_id: payload.token_id,
        },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    let mint_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );
    let associated_addr =
        spl_associated_token_account::get_associated_token_address(&to_authority, &mint_key);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(associated_addr, false),
            AccountMeta::new_readonly(to_authority, false),
            AccountMeta::new(mint_key, false),
            AccountMeta::new(mint_meta_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
            AccountMeta::new_readonly(spl_associated_token_account::id(), false),
            AccountMeta::new_readonly(spl_token_metadata::id(), false),
        ],
        data: (crate::instruction::Instruction::CompleteWrapped, data).try_to_vec()?,
    })
}

pub fn complete_wrapped_meta(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadTransfer,
    data: CompleteWrappedMetaData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, _claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
            token_id: payload.token_id,
        },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    let mint_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );
    // SPL Metadata
    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData { mint: mint_key },
        &spl_token_metadata::id(),
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new_readonly(mint_key, false),
            AccountMeta::new_readonly(mint_meta_key, false),
            AccountMeta::new(spl_metadata, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
            AccountMeta::new_readonly(spl_associated_token_account::id(), false),
            AccountMeta::new_readonly(spl_token_metadata::id(), false),
        ],
        data: (crate::instruction::Instruction::CompleteWrappedMeta, data).try_to_vec()?,
    })
}

pub fn register_chain(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadGovernanceRegisterChain,
    data: RegisterChainData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa);
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: payload.chain,
            emitter_address: payload.endpoint_address,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(endpoint, false),
            message_acc,
            claim_acc,
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: (crate::instruction::Instruction::RegisterChain, data).try_to_vec()?,
    })
}

fn claimable_vaa(
    bridge_id: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
) -> (AccountMeta, AccountMeta) {
    let claim_key = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: vaa.emitter_address,
            emitter_chain: vaa.emitter_chain,
            sequence: vaa.sequence,
        },
        &bridge_id,
    );

    (
        AccountMeta::new_readonly(message_key, false),
        AccountMeta::new(claim_key, false),
    )
}

pub fn transfer_native(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    mint: Pubkey,
    data: TransferNativeData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );

    let authority_signer_key = AuthoritySigner::key(None, &program_id);
    let custody_signer_key = CustodySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // SPL Metadata
    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData { mint },
        &spl_token_metadata::id(),
    );

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new(mint, false),
            AccountMeta::new_readonly(spl_metadata, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(authority_signer_key, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (crate::instruction::Instruction::TransferNative, data).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn transfer_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    from_owner: Pubkey,
    token_chain: u16,
    token_address: ForeignAddress,
    token_id: U256,
    data: TransferWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);

    let wrapped_mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
            token_id,
        },
        &program_id,
    );
    let wrapped_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData {
            mint_key: wrapped_mint_key,
        },
        &program_id,
    );

    let authority_signer = AuthoritySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // SPL Metadata
    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData {
            mint: wrapped_mint_key,
        },
        &spl_token_metadata::id(),
    );

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new_readonly(from_owner, true),
            AccountMeta::new(wrapped_mint_key, false),
            AccountMeta::new_readonly(wrapped_meta_key, false),
            AccountMeta::new_readonly(spl_metadata, false),
            AccountMeta::new_readonly(authority_signer, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (crate::instruction::Instruction::TransferWrapped, data).try_to_vec()?,
    })
}

pub fn upgrade_contract(
    program_id: Pubkey,
    payer: Pubkey,
    payload_message: Pubkey,
    emitter: Pubkey,
    new_contract: Pubkey,
    spill: Pubkey,
    sequence: u64,
) -> Instruction {
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    let (upgrade_authority, _) = Pubkey::find_program_address(&["upgrade".as_bytes()], &program_id);

    let (program_data, _) = Pubkey::find_program_address(
        &[program_id.as_ref()],
        &solana_program::bpf_loader_upgradeable::id(),
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(payload_message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new_readonly(upgrade_authority, false),
            AccountMeta::new(spill, false),
            AccountMeta::new(new_contract, false),
            AccountMeta::new(program_data, false),
            AccountMeta::new(program_id, false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            AccountMeta::new_readonly(solana_program::bpf_loader_upgradeable::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (
            crate::instruction::Instruction::UpgradeContract,
            UpgradeContractData {},
        )
            .try_to_vec()
            .unwrap(),
    }
}
