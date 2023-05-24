## Design

Foreign NFT -> Sui
- Mint a wrapped representation using the Minter module (similar to what marketplaces to)

Sui native NFT -> another chain
- Get coin metadata in Display, emit a WH message

Questions
- Can multiple display objects be created using the same Publisher and for the same type? Publisher is unique to module but not unique to package. Is Publisher unique to a type?
- Can check if a Publisher corresponds to a type using: package::from_module