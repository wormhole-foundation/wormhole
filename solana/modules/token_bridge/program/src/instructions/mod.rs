use crate::{
    accounts::{
        AuthoritySigner, ConfigAccount, CustodyAccount, CustodyAccountDerivationData,
        CustodySigner, EmitterAccount, Endpoint, EndpointDerivationData, MintSigner,
        WrappedDerivationData, WrappedMint, WrappedTokenMeta,
    },
    api::{
        complete_transfer::{CompleteNativeData, CompleteWrappedData},
        AttestTokenData, CreateWrappedData, RegisterChainData, TransferNativeData,
        TransferWrappedData,
    },
    messages::{PayloadAssetMeta, PayloadGovernanceRegisterChain, PayloadTransfer},
};
use borsh::BorshSerialize;
use bridge::vaa::SerializePayload;
use bridge::{
    accounts::{
        Bridge, Claim, ClaimDerivationData, FeeCollector, Message, MessageDerivationData, Sequence,
        SequenceDerivationData,
    },
    api::ForeignAddress,
    types::{BridgeConfig, PostedMessage},
    vaa::{ClaimableVAA, PayloadMessage},
};
use primitive_types::U256;
use solana_program::instruction::Instruction;
use solitaire::{processors::seeded::Seeded, AccountState};
use solitaire_client::{AccountMeta, Keypair, Pubkey};
use spl_token::state::Mint;

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
        data: crate::instruction::Instruction::Initialize(bridge).try_to_vec()?,
    })
}

pub fn complete_native(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    message: PostedMessage,
    to: Pubkey,
    mint: Pubkey,
    data: CompleteNativeData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(bridge_id, message_key, message.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: message.emitter_chain,
            emitter_address: message.emitter_address,
        },
        &program_id,
    );
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );
    let custody_signer_key = CustodySigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: crate::instruction::Instruction::CompleteNative(data).try_to_vec()?,
    })
}

pub fn complete_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    message: PostedMessage,
    payload: PayloadTransfer,
    to: Pubkey,
    data: CompleteWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(bridge_id, message_key, message.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: message.emitter_chain,
            emitter_address: message.emitter_address,
        },
        &bridge_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            AccountMeta::new_readonly(mint_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: crate::instruction::Instruction::CompleteWrapped(data).try_to_vec()?,
    })
}

pub fn create_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    message: PostedMessage,
    payload: PayloadAssetMeta,
    data: CreateWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(bridge_id, message_key, message.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: message.emitter_chain,
            emitter_address: message.emitter_address,
        },
        &bridge_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let mint_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            AccountMeta::new_readonly(endpoint, false),
            message_acc,
            claim_acc,
            AccountMeta::new(mint_key, false),
            AccountMeta::new(mint_meta_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: crate::instruction::Instruction::CreateWrapped(data).try_to_vec()?,
    })
}

pub fn register_chain(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    message: PostedMessage,
    payload: PayloadGovernanceRegisterChain,
    data: RegisterChainData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(bridge_id, message_key, message.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: message.emitter_chain,
            emitter_address: message.emitter_address,
        },
        &bridge_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            AccountMeta::new_readonly(endpoint, false),
            message_acc,
            claim_acc,
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: crate::instruction::Instruction::RegisterChain(data).try_to_vec()?,
    })
}

fn claimable_vaa(
    bridge_id: Pubkey,
    message_key: Pubkey,
    message: PostedMessage,
) -> (AccountMeta, AccountMeta) {
    let claim_key = Claim::<'_, { AccountState::Initialized }>::key(
        &ClaimDerivationData {
            emitter_address: message.emitter_address,
            emitter_chain: message.emitter_chain,
            sequence: message.sequence,
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

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let payload = PayloadTransfer {
        amount: U256::from(data.amount),
        token_address: mint.to_bytes(),
        token_chain: 1,
        to: data.target_address,
        to_chain: data.target_chain,
        fee: U256::from(data.fee),
    };
    let message_key = Message::<'_, { AccountState::Uninitialized }>::key(
        &MessageDerivationData {
            emitter_key: emitter_key.to_bytes(),
            emitter_chain: 1,
            nonce: data.nonce,
            payload: payload.try_to_vec().unwrap(),
            sequence: None,
        },
        &bridge_id,
    );
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
            AccountMeta::new(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new(mint, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(authority_signer_key, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            AccountMeta::new_readonly(bridge_config, false),
            AccountMeta::new(message_key, false),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: crate::instruction::Instruction::TransferNative(data).try_to_vec()?,
    })
}

pub fn transfer_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    from: Pubkey,
    from_owner: Pubkey,
    token_chain: u16,
    token_address: ForeignAddress,
    data: TransferWrappedData,
    sequence: u64,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);

    let wrapped_mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
        },
        &program_id,
    );
    let wrapped_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
        },
        &program_id,
    );

    let mint_authority_key = MintSigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let payload = PayloadTransfer {
        amount: U256::from(data.amount),
        token_address,
        token_chain,
        to: data.target_address,
        to_chain: data.target_chain,
        fee: U256::from(data.fee),
    };
    let message_key = Message::<'_, { AccountState::Uninitialized }>::key(
        &MessageDerivationData {
            emitter_key: emitter_key.to_bytes(),
            emitter_chain: 1,
            nonce: data.nonce,
            payload: payload.try_to_vec().unwrap(),
            sequence: None,
        },
        &bridge_id,
    );
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
            AccountMeta::new(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new(from_owner, true),
            AccountMeta::new_readonly(wrapped_mint_key, false),
            AccountMeta::new_readonly(wrapped_meta_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            AccountMeta::new_readonly(bridge_config, false),
            AccountMeta::new(message_key, false),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: crate::instruction::Instruction::TransferWrapped(data).try_to_vec()?,
    })
}

pub fn attest(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    mint: Pubkey,
    mint_data: Mint,
    mint_meta: Pubkey,
    nonce: u32,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let payload = PayloadAssetMeta {
        token_address: mint.to_bytes(),
        token_chain: 1,
        decimals: U256::from(mint_data.decimals),
        symbol: "".to_string(), // TODO metadata
        name: "".to_string(),
    };
    let message_key = Message::<'_, { AccountState::Uninitialized }>::key(
        &MessageDerivationData {
            emitter_key: emitter_key.to_bytes(),
            emitter_chain: 1,
            nonce,
            sequence: None,
            payload: payload.try_to_vec().unwrap(),
        },
        &bridge_id,
    );
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
            AccountMeta::new(config_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(mint_meta, false),
            // Bridge accounts
            AccountMeta::new_readonly(bridge_config, false),
            AccountMeta::new(message_key, false),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: crate::instruction::Instruction::AttestToken(AttestTokenData { nonce })
            .try_to_vec()?,
    })
}
