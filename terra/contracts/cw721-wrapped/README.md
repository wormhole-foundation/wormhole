# CW721 Metadata Onchain

NFT creators may want to store their NFT metadata on-chain so other contracts are able to interact with it.
With CW721-Base in CosmWasm, we allow you to store any data on chain you wish, using a generic `extension: T`.

In order to support on-chain metadata, and to demonstrate how to use the extension ability, we have created this simple contract.
There is no business logic here, but looking at `lib.rs` will show you how do define custom data that is included when minting and
available in all queries.

In particular, here we define:

```rust
#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema, Debug)]
pub struct Trait {
    pub display_type: Option<String>,
    pub trait_type: String,
    pub value: String,
}

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema, Debug)]
pub struct Metadata {
    pub image: Option<String>,
    pub image_data: Option<String>,
    pub external_url: Option<String>,
    pub description: Option<String>,
    pub name: Option<String>,
    pub attributes: Option<Vec<Trait>>,
    pub background_color: Option<String>,
    pub animation_url: Option<String>,
    pub youtube_url: Option<String>,
}

pub type Extension = Option<Metadata>;
```

In particular, the fields defined conform to the properties supported in the [Opensea Metadata Standard](https://docs.opensea.io/docs/metadata-standards).


This means when you query `NftInfo{name: "Enterprise"}`, you will get something like:

```json
{
  "name": "Enterprise",
  "token_uri": "https://starships.example.com/Starship/Enterprise.json",
  "extension": {
    "image": null,
    "image_data": null,
    "external_url": null,
    "description": "Spaceship with Warp Drive",
    "name": "Starship USS Enterprise",
    "attributes": null,
    "background_color": null,
    "animation_url": null,
    "youtube_url": null
  }
}
```

Please look at the test code for an example usage in Rust.

## Notice

Feel free to use this contract out of the box, or as inspiration for further customization of cw721-base.
We will not be adding new features or business logic here.
