use dep::aztec::macros::aztec;
mod structs;

#[aztec]
pub contract Wormhole {
    use crate::structs::{Provider, Guardian, WormholeStorage};

    use dep::aztec::{
        macros::{
            functions::{initializer, public},
            storage::storage,
        },
        prelude::{Map, PublicMutable, AztecAddress},
        protocol_types::traits::ToField
    };

    // use dep::token::Token; 

    #[storage]
    struct Storage<Context> {
        state: PublicMutable<WormholeStorage, Context>,
        sequences: Map<AztecAddress, PublicMutable<u64, Context>, Context>,
        wormhole_address: PublicMutable<AztecAddress, Context>,
        token_address: PublicMutable<AztecAddress, Context>,
        guardian_1: Map<u32, PublicMutable<Guardian, Context>, Context>, // FIXED 13 guardians
        guardian_2: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_3: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_4: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_5: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_6: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_7: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_8: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_9: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_10: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_11: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_12: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_13: Map<u32, PublicMutable<Guardian, Context>, Context>,
        current_guardian_set_index: Map<u32, PublicMutable<u64, Context>, Context>, // map index to expiration time
    }

    #[public]
    #[initializer]
    fn init(
        chain_id: u16,
        evm_chain_id: u16,
        wormhole_address: AztecAddress, // TODO: WHO IS THIS? WHERE DO WE GET THE DEPOSIT ADDRESS? WHAT TOKEN ARE WE SENDING?
        // token_address: AztecAddress, // TODO: DEPLOY THIS BEFORE DEPLOYING WORMHOLE (MUST EXIST ALREADY)
    ) {
        let provider: Provider = Provider {
            chain_id,
            evm_chain_id,
        };
        // Initialize the wormhole state storage
        storage.state.write(WormholeStorage::init(provider));

        // Define the addresses for the wormhole account and token contract
        storage.wormhole_address.write(wormhole_address);
        // storage.token_address.write(token_address);
        storage.current_guardian_set_index.at(0).write(18_446_744_073_709_551_615); // 2^64 - 1
    }

    #[public]
    fn set_guardian1(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_1.at(index).write(guardian);
    }

    #[public]
    fn set_guardian2(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_2.at(index).write(guardian);
    }

    #[public]
    fn set_guardian3(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_3.at(index).write(guardian);
    }

    #[public]
    fn set_guardian4(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_4.at(index).write(guardian);
    }

    #[public]
    fn set_guardian5(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_5.at(index).write(guardian);
    }

    #[public]
    fn set_guardian6(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_6.at(index).write(guardian);
    }

    #[public]
    fn set_guardian7(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_7.at(index).write(guardian);
    }

    #[public]
    fn set_guardian8(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_8.at(index).write(guardian);
    }

    #[public]
    fn set_guardian9(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_9.at(index).write(guardian);
    }

    #[public]
    fn set_guardian10(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_10.at(index).write(guardian);
    }

    #[public]
    fn set_guardian11(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_11.at(index).write(guardian);
    }

    #[public]
    fn set_guardian12(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_12.at(index).write(guardian);
    }

    #[public]
    fn set_guardian13(
        guardian: Guardian,
        index: u32,
    ){ 
        storage.guardian_13.at(index).write(guardian);
    }

    #[public]
    fn expire_guardian_set(
        index: u32,
    ) {
        storage.current_guardian_set_index.at(index).write(context.timestamp() + 86400);
    }

    #[public]
    fn guardian_set_expired(
        index: u32,
    ) -> bool {
        let timestamp = storage.current_guardian_set_index.at(index).read();
        context.timestamp() > timestamp
    }

    #[public]
    fn publish_message(
        nonce: u64,
        payloads: [[u8; 31]; 8], // size of payload needs investigation, looks like max size is Field's size unless we send multiple logs.
        message_fee: u128,
        consistency: u8,
    ) -> u64 {
        // check fee
        assert(storage.state.read().message_fee <= message_fee, "insufficient fee");

        // increase current sequence
        let sequence = storage.sequences.at(context.msg_sender()).read();
        storage.sequences.at(context.msg_sender()).write(sequence + 1);

        // prepare message
        let msg = [
            context.msg_sender().to_field(), 
            sequence as Field,  
            nonce as Field,
            consistency as Field, 
            context.timestamp() as Field,
            Field::from_le_bytes(payloads[0]),
            Field::from_le_bytes(payloads[1]),
            Field::from_le_bytes(payloads[2]),
            Field::from_le_bytes(payloads[3]),
            Field::from_le_bytes(payloads[4]),
            Field::from_le_bytes(payloads[5]),
            Field::from_le_bytes(payloads[6]),
            Field::from_le_bytes(payloads[7]),
        ];

        // let receiver_address = storage.wormhole_address.read();
        // let token_address = storage.token_address.read();

        // // deposit the message fee to wormhole token contract
        // let _ = Token::at(token_address).transfer_in_public(
        //     context.msg_sender(),
        //     receiver_address, 
        //     message_fee,
        //     sequence as Field,
        // ).call(&mut context);

        context.emit_public_log(msg);

        sequence
    }

    #[public]
    fn set_provider(provider: Provider) {
        WormholeStorage::set_provider(storage.state.read(),provider);
    }

    #[public]
    fn get_provider() -> Provider {
        WormholeStorage::get_provider(storage.state.read())
    }

    // TODO: E2E tests with TXE: https://docs.aztec.network/developers/guides/smart_contracts/testing
}