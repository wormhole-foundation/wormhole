import { TokenTypes } from "aptos";

export type TokenBridgeState = {
  consumed_vaas: {
    elems: {
      handle: string;
    };
  };
  emitter_cap: {
    emitter: string;
    sequence: string;
  };
  governance_chain_id: {
    number: string;
  };
  governance_contract: {
    external_address: string;
  };
  native_infos: {
    handle: string;
  };
  registered_emitters: {
    handle: string;
  };
  signer_cap: {
    account: string;
  };
  wrapped_infos: {
    handle: string;
  };
};

export type OriginInfo = {
  token_address: {
    external_address: string;
  };
  token_chain: {
    number: string; // lol
  };
};

export type NftBridgeState = {
  consumed_vaas: {
    elems: {
      handle: string;
    };
  };
  emitter_cap: {
    emitter: string;
    sequence: string;
  };
  native_infos: {
    handle: string;
  };
  registered_emitters: {
    handle: string;
  };
  signer_cap: {
    account: string;
  };
  spl_cache: {
    handle: string;
  };
  wrapped_infos: {
    handle: string;
  };
};

export type CreateTokenDataEvent = {
  version: string;
  guid: {
    creation_number: string;
    account_address: string;
  };
  sequence_number: string;
  type: "0x3::token::CreateTokenDataEvent";
  data: {
    description: string;
    id: TokenTypes.TokenDataId;
    maximum: string;
    mutability_config: {
      description: boolean;
      maximum: boolean;
      properties: boolean;
      royalty: boolean;
      uri: boolean;
    };
    name: string;
    property_keys: [string];
    property_types: [string];
    property_values: [string];
    royalty_payee_address: string;
    royalty_points_denominator: string;
    royalty_points_numerator: string;
    uri: string;
  };
};

export type DepositEvent = {
  version: string;
  guid: {
    creation_number: string;
    account_address: string;
  };
  sequence_number: string;
  type: "0x3::token::DepositEvent";
  data: {
    amount: string;
    id: TokenTypes.TokenId;
  };
};
