mod structs;

// docs:start:empty-contract
use dep::aztec::macros::aztec;

#[aztec]
pub contract Wormhole {
    // docs:end:empty-contract
    use crate::structs::{Provider, GuardianSet};

    // docs:start:all-deps
    use dep::aztec::{
        macros::{
            functions::{initializer, internal, private, public},
            storage::storage,
        },
        prelude::{Map, AztecAddress, PublicMutable, PrivateMutable},
    };

    use dep::aztec::protocol_types::{
        storage::map::derive_storage_slot_in_map,
        traits::ToField,
    };

    #[storage]
    struct Storage<Context> {
        provider: PublicMutable<Provider, Context>,
        guardian_sets: Map<u32, PublicMutable<GuardianSet, Context>, Context>,
        guardian_set_index: PublicMutable<u32, Context>,
        guardian_set_expiry: PublicMutable<u32, Context>,
        sequences: Map<AztecAddress, PublicMutable<Field, Context>, Context>,
        consumed_governance_actions: Map<Field, PublicMutable<bool, Context>, Context>,
        initialized_implementations: Map<Field, PublicMutable<bool, Context>, Context>,
        message_fee: PublicMutable<Field, Context>,
        evm_chain_id: PublicMutable<Field, Context>
    }

    #[public]
    fn set_provider(a: Provider) {
        storage.provider.write(a);
    }

    #[public]
    fn get_provider() -> Provider {
        storage.provider.read()
    }

    #[public]
    fn add_guardian_set(to: u32, gs: GuardianSet) -> Field {
        storage.guardian_sets.at(to).write(gs);
        // returns storage slot for key
        derive_storage_slot_in_map(storage.guardian_sets.storage_slot, to)
    }

    #[public]
    fn get_guardian_sets(address: u32) -> GuardianSet {
        storage.guardian_sets.at(address).read()
    }

    #[public]
    fn expire_guardian_set(index: u32, current_timestamp: u32) -> Field {
        // Get the guardian set
        let mut guardian_set = storage.guardian_sets.at(index).read();
        
        // Update the expiration time
        guardian_set.expiration_time = current_timestamp + 86400;
        
        // Write the updated guardian set back to storage
        storage.guardian_sets.at(index).write(guardian_set);
        
        // Return storage slot for key
        derive_storage_slot_in_map(storage.guardian_sets.storage_slot, index)
    }

    #[public]
    fn set_guardian_set_index(a: u32) {
        storage.guardian_set_index.write(a);
    }

    #[public]
    fn get_guardian_set_index() -> u32 {
        storage.guardian_set_index.read()
    }

    #[public]
    fn set_guardian_set_expiry(a: u32) {
        storage.guardian_set_expiry.write(a);
    }

    #[public]
    fn get_guardian_set_expiry() -> u32 {
        storage.guardian_set_expiry.read()
    }

    #[public]
    fn add_sequence(to: AztecAddress, sequence: Field) -> Field {
        storage.sequences.at(to).write(sequence);
        // returns storage slot for key
        derive_storage_slot_in_map(storage.sequences.storage_slot, to)
    }

    #[public]
    fn get_sequences(address: AztecAddress) -> Field {
        storage.sequences.at(address).read()
    }

    #[public]
    fn add_consumed_governance_action(to: Field, cga: bool) -> Field {
        storage.consumed_governance_actions.at(to).write(cga);
        // returns storage slot for key
        derive_storage_slot_in_map(storage.consumed_governance_actions.storage_slot, to)
    }

    #[public]
    fn get_consumed_governance_actions(address: Field) -> bool {
        storage.consumed_governance_actions.at(address).read()
    }

    #[public]
    fn set_initialized_implementation(to: Field, ii: bool) -> Field {
        storage.initialized_implementations.at(to).write(ii);
        // returns storage slot for key
        derive_storage_slot_in_map(storage.initialized_implementations.storage_slot, to)
    }

    #[public]
    fn get_initialized_implementations(address: Field) -> bool {
        storage.initialized_implementations.at(address).read()
    }

    #[public]
    fn set_message_fee(a: Field) {
        storage.message_fee.write(a);
    }

    #[public]
    fn get_message_fee() -> Field {
        storage.message_fee.read()
    }

    #[public]
    fn set_evm_chain_id(a: Field) {
        storage.evm_chain_id.write(a);
    }

    #[public]
    fn get_evm_chain_id() -> Field {
        storage.evm_chain_id.read()
    }

    // Publish a message to be attested by the Wormhole network
    #[public]
    fn publishMessage(
        nonce: u32,
        payload: [u8; 24], // size of payload needs investigation, looks like max size is Field's size unless we send multiple logs.
        consistencyLevel: u8) {
        // check fee
        assert(context.transaction_fee() == get_message_fee(), "invalid fee");

        let sequence = useSequence(context.msg_sender());
        
        // Create an array of Field elements
        let message = [
            context.msg_sender().to_field(),
            sequence,  // Already a Field
            Field::from(nonce),
            Field::from(consistencyLevel),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload),
            Field::from_be_bytes(payload)
        ];
        
        // emit log
        context.emit_public_log(message);
    }

    #[public]
    fn useSequence(emitter: AztecAddress) -> Field {
        let sequence = get_sequences(emitter);
        add_sequence(emitter, sequence + 1);
        sequence
    }

}