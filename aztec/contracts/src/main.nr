use dep::aztec::macros::aztec;
mod structs;

/**

To deploy on testnet, use the following commands (following the tutorial):
1. Set the environment variable for the node URL:
export NODE_URL=https://aztec-alpha-testnet-fullnode.zkv.xyz
export SPONSORED_FPC_ADDRESS=0x1260a43ecf03e985727affbbe3e483e60b836ea821b6305bea1c53398b986047

2. Create a new wallets:
2a. Create a new wallet for the owner:
aztec-wallet create-account \
    --register-only \
    --node-url $NODE_URL \
    --alias owner-wallet
2b. Create a new wallet for the receiver:
aztec-wallet create-account \
    --register-only \
    --node-url $NODE_URL \
    --alias receiver-wallet

3. Register owner account with FPC:
3a. Register the owner wallet with the FPC:
aztec-wallet register-contract \
    --node-url $NODE_URL \
    --from owner-wallet \
    --alias sponsoredfpc \
    $SPONSORED_FPC_ADDRESS SponsoredFPC \
    --salt 0
3b. Register the receiver wallet with the FPC:
aztec-wallet register-contract \
    --node-url $NODE_URL \
    --from receiver-wallet \
    --alias sponsoredfpc \
    $SPONSORED_FPC_ADDRESS SponsoredFPC \
    --salt 0

4. Deploy accounts (may get error `Timeout awaiting isMined` but that is fine continue with the next step):
4a. Deploy the owner wallet:
aztec-wallet deploy-account \
    --node-url $NODE_URL \
    --from owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --register-class
4b. Deploy the receiver wallet:
aztec-wallet deploy-account \
    --node-url $NODE_URL \
    --from receiver-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --register-class

5. Deploy the Token contract (wait for it to be mined on testnet check txhash at http://aztecscan.xyz/):
aztec-wallet deploy \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --alias token \
    TokenContract \
    --args accounts:owner-wallet WormToken WORM 18 --no-wait

6. Mint tokens to public & private now to call the upcoming functions:
6a. Mint public tokens:


7. Deploy the Wormhole contract (wait for it to be mined on testnet check txhash at http://aztecscan.xyz/):
aztec-wallet deploy \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --alias wormhole \
    Wormhole \
    --args 13 13 *receiver address* *token contract address* --no-wait

*/

#[aztec]
pub contract Wormhole {
    use crate::structs::{Provider, Guardian, WormholeStorage};

    use dep::aztec::{
        macros::{
            functions::{initializer, public, private, internal},
            storage::storage,
        },
        prelude::{Map, PublicMutable, AztecAddress},
        protocol_types::traits::ToField
    };

    use dep::token::Token; 

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
        wormhole_address: AztecAddress, 
        token_address: AztecAddress, 
    ) {
        let provider: Provider = Provider {
            chain_id,
            evm_chain_id,
        };
        // Initialize the wormhole state storage
        storage.state.write(WormholeStorage::init(provider));

        // Define the addresses for the wormhole account and token contract
        storage.wormhole_address.write(wormhole_address);
        storage.token_address.write(token_address);
        storage.current_guardian_set_index.at(0).write(18_446_744_073_709_551_615); // 2^64 - 1
    }

    #[public]
    fn publish_message_in_public(
        nonce: u64,
        payloads: [[u8; 31]; 8], 
        message_fee: u128,
        consistency: u8,
        from: AztecAddress, // explicitly pass token sender address (leaks privacy)  TODO: proper UTXO workflow
        token_nonce: Field,
    ) -> u64 {
        // check fee
        assert(storage.state.read().message_fee <= message_fee, "insufficient fee");

        let receiver_address = storage.wormhole_address.read();
        let token_address = storage.token_address.read();

        // deposit the message fee to wormhole token contract
        let _ = Token::at(token_address).transfer_in_public(
            from,
            receiver_address, 
            message_fee,
            token_nonce,  
        ).call(&mut context);

        // 16 as u64
        // increase current sequence
        let sequence = storage.sequences.at(context.msg_sender()).read();
        storage.sequences.at(context.msg_sender()).write(sequence + 1);

        // prepare message
        let msg: [Field; 13] = [
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

        context.emit_public_log(msg);

        sequence
    }

    #[private]
    fn publish_message_in_private(
        nonce: u64,
        payloads: [[u8; 31]; 8], 
        message_fee: u128,
        consistency: u8,
        from: AztecAddress, 
        token_nonce: Field,
    ) {
        // hard coded message fee, receiver address and token address (avoid using shared state)
        // check fee
        assert(1 <= message_fee, "insufficient fee");

        let receiver_address = AztecAddress {
            inner: 0x1f41267c06dae96c9c3906c5f77cbc28602cc70d6d7e9d2c3072cb0a5b13edd2 // TODO: change this each time
        };
        let token_address = AztecAddress {
            inner: 0x15edee5716556b6d29a9893c9d6872117709a7bf23fb82a72ef6bf7de1de9da0 // TODO: change this each time
        };

        // deposit the message fee to wormhole token contract
        let _ = Token::at(token_address).transfer_in_private(
            from,
            receiver_address, 
            message_fee,
            token_nonce,  
        ).call(&mut context);

        // enqueue publish message
        Wormhole::at(context.this_address())._publish_message(
            nonce, 
            payloads, 
            consistency
        ).enqueue(&mut context);
    }

    #[public]
    #[internal]
    fn _publish_message(
        nonce: u64,
        payloads: [[u8; 31]; 8], 
        consistency: u8,
    ) -> u64 {
        // increase current sequence
        let sequence = storage.sequences.at(context.msg_sender()).read();
        storage.sequences.at(context.msg_sender()).write(sequence + 1);

        // prepare message
        let msg: [Field; 13] = [
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

        context.emit_public_log(msg);

        sequence
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
    fn set_provider(provider: Provider) {
        WormholeStorage::set_provider(storage.state.read(),provider);
    }

    #[public]
    fn get_provider() -> Provider {
        WormholeStorage::get_provider(storage.state.read())
    }

    // TODO: E2E tests with TXE: https://docs.aztec.network/developers/guides/smart_contracts/testing
}