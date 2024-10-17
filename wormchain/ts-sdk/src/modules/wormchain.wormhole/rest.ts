//@ts-nocheck
/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface ProtobufAny {
  "@type"?: string;
}

export interface RpcStatus {
  /** @format int32 */
  code?: number;
  message?: string;
  details?: ProtobufAny[];
}

/**
* message SomeRequest {
         Foo some_parameter = 1;
         PageRequest pagination = 2;
 }
*/
export interface V1Beta1PageRequest {
  /**
   * key is a value returned in PageResponse.next_key to begin
   * querying the next page most efficiently. Only one of offset or key
   * should be set.
   * @format byte
   */
  key?: string;

  /**
   * offset is a numeric offset that can be used when key is unavailable.
   * It is less efficient than using key. Only one of offset or key should
   * be set.
   * @format uint64
   */
  offset?: string;

  /**
   * limit is the total number of results to be returned in the result page.
   * If left empty it will default to a value to be set by each app.
   * @format uint64
   */
  limit?: string;

  /**
   * count_total is set to true  to indicate that the result set should include
   * a count of the total number of items available for pagination in UIs.
   * count_total is only respected when offset is used. It is ignored when key
   * is set.
   */
  count_total?: boolean;

  /**
   * reverse is set to true if results are to be returned in the descending order.
   *
   * Since: cosmos-sdk 0.43
   */
  reverse?: boolean;
}

/**
* PageResponse is to be embedded in gRPC response messages where the
corresponding request message has used PageRequest.

 message SomeResponse {
         repeated Bar results = 1;
         PageResponse page = 2;
 }
*/
export interface V1Beta1PageResponse {
  /**
   * next_key is the key to be passed to PageRequest.key to
   * query the next page most efficiently. It will be empty if
   * there are no more results.
   * @format byte
   */
  next_key?: string;

  /**
   * total is total number of results available if PageRequest.count_total
   * was set, its value is undefined otherwise
   * @format uint64
   */
  total?: string;
}

export interface WormchainwormholeConfig {
  /** @format uint64 */
  guardian_set_expiration?: string;

  /** @format byte */
  governance_emitter?: string;

  /** @format int64 */
  governance_chain?: number;

  /** @format int64 */
  chain_id?: number;
}

export interface WormchainwormholeConsensusGuardianSetIndex {
  /** @format int64 */
  index?: number;
}

export interface WormchainwormholeGuardianSet {
  /** @format int64 */
  index?: number;
  keys?: string[];

  /** @format uint64 */
  expirationTime?: string;
}

export interface WormchainwormholeGuardianValidator {
  /** @format byte */
  guardianKey?: string;

  /** @format byte */
  validatorAddr?: string;
}

export interface WormchainwormholeReplayProtection {
  index?: string;
}

export interface WormchainwormholeSequenceCounter {
  index?: string;

  /** @format uint64 */
  sequence?: string;
}

export type WormholeEmptyResponse = object;

export type WormholeMsgAllowlistResponse = object;

export type WormholeMsgExecuteGovernanceVAAResponse = object;

export interface WormholeMsgInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address?: string;

  /**
   * Data contains base64-encoded bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

/**
 * MsgMigrateContractResponse returns contract migration result data.
 */
export interface WormholeMsgMigrateContractResponse {
  /**
   * Data contains same raw bytes returned as data from the wasm contract.
   * (May be empty)
   * @format byte
   */
  data?: string;
}

export type WormholeMsgRegisterAccountAsGuardianResponse = object;

export interface WormholeMsgStoreCodeResponse {
  /**
   * CodeID is the reference to the stored WASM code
   * @format uint64
   */
  code_id?: string;

  /**
   * Checksum is the sha256 hash of the stored code
   * @format byte
   */
  checksum?: string;
}

export type WormholeMsgWasmInstantiateAllowlistResponse = object;

export interface WormholeQueryAllGuardianSetResponse {
  GuardianSet?: WormchainwormholeGuardianSet[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryAllGuardianValidatorResponse {
  guardianValidator?: WormchainwormholeGuardianValidator[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryAllReplayProtectionResponse {
  replayProtection?: WormchainwormholeReplayProtection[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryAllSequenceCounterResponse {
  sequenceCounter?: WormchainwormholeSequenceCounter[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryAllValidatorAllowlistResponse {
  allowlist?: WormholeValidatorAllowedAddress[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryAllWasmInstantiateAllowlistResponse {
  allowlist?: WormholeWasmInstantiateAllowedContractCodeId[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeQueryGetConfigResponse {
  Config?: WormchainwormholeConfig;
}

export interface WormholeQueryGetConsensusGuardianSetIndexResponse {
  ConsensusGuardianSetIndex?: WormchainwormholeConsensusGuardianSetIndex;
}

export interface WormholeQueryGetGuardianSetResponse {
  GuardianSet?: WormchainwormholeGuardianSet;
}

export interface WormholeQueryGetGuardianValidatorResponse {
  guardianValidator?: WormchainwormholeGuardianValidator;
}

export interface WormholeQueryGetReplayProtectionResponse {
  replayProtection?: WormchainwormholeReplayProtection;
}

export interface WormholeQueryGetSequenceCounterResponse {
  sequenceCounter?: WormchainwormholeSequenceCounter;
}

export interface WormholeQueryIbcComposabilityMwContractResponse {
  contractAddress?: string;
}

export interface WormholeQueryLatestGuardianSetIndexResponse {
  /** @format int64 */
  latestGuardianSetIndex?: number;
}

export interface WormholeQueryValidatorAllowlistResponse {
  validator_address?: string;
  allowlist?: WormholeValidatorAllowedAddress[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface WormholeValidatorAllowedAddress {
  /** the validator/guardian that controls this entry */
  validator_address?: string;

  /** the allowlisted account */
  allowed_address?: string;

  /** human readable name */
  name?: string;
}

export interface WormholeWasmInstantiateAllowedContractCodeId {
  /** bech32 address of the contract that can call wasm instantiate without a VAA */
  contract_address?: string;

  /**
   * reference to the stored WASM code that can be instantiated
   * @format uint64
   */
  code_id?: string;
}

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, ResponseType } from "axios";

export type QueryParamsType = Record<string | number, any>;

export interface FullRequestParams extends Omit<AxiosRequestConfig, "data" | "params" | "url" | "responseType"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseType;
  /** request body */
  body?: unknown;
}

export type RequestParams = Omit<FullRequestParams, "body" | "method" | "query" | "path">;

export interface ApiConfig<SecurityDataType = unknown> extends Omit<AxiosRequestConfig, "data" | "cancelToken"> {
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<AxiosRequestConfig | void> | AxiosRequestConfig | void;
  secure?: boolean;
  format?: ResponseType;
}

export enum ContentType {
  Json = "application/json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
}

export class HttpClient<SecurityDataType = unknown> {
  public instance: AxiosInstance;
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private secure?: boolean;
  private format?: ResponseType;

  constructor({ securityWorker, secure, format, ...axiosConfig }: ApiConfig<SecurityDataType> = {}) {
    this.instance = axios.create({ ...axiosConfig, baseURL: axiosConfig.baseURL || "" });
    this.secure = secure;
    this.format = format;
    this.securityWorker = securityWorker;
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  private mergeRequestParams(params1: AxiosRequestConfig, params2?: AxiosRequestConfig): AxiosRequestConfig {
    return {
      ...this.instance.defaults,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.instance.defaults.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  private createFormData(input: Record<string, unknown>): FormData {
    return Object.keys(input || {}).reduce((formData, key) => {
      const property = input[key];
      formData.append(
        key,
        property instanceof Blob
          ? property
          : typeof property === "object" && property !== null
          ? JSON.stringify(property)
          : `${property}`,
      );
      return formData;
    }, new FormData());
  }

  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams): Promise<AxiosResponse<T>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const responseFormat = (format && this.format) || void 0;

    if (type === ContentType.FormData && body && body !== null && typeof body === "object") {
      requestParams.headers.common = { Accept: "*/*" };
      requestParams.headers.post = {};
      requestParams.headers.put = {};

      body = this.createFormData(body as Record<string, unknown>);
    }

    return this.instance.request({
      ...requestParams,
      headers: {
        ...(type && type !== ContentType.FormData ? { "Content-Type": type } : {}),
        ...(requestParams.headers || {}),
      },
      params: query,
      responseType: responseFormat,
      data: body,
      url: path,
    });
  };
}

/**
 * @title wormchain/wormhole/config.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryAllowlistAll
   * @request GET:/wormhole_foundation/wormchain/wormhole/allowlist
   */
  queryAllowlistAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllValidatorAllowlistResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/allowlist`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryAllowlist
   * @request GET:/wormhole_foundation/wormchain/wormhole/allowlist/{validator_address}
   */
  queryAllowlist = (
    validatorAddress: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryValidatorAllowlistResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/allowlist/${validatorAddress}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryConfig
   * @summary Queries a config by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/config
   */
  queryConfig = (params: RequestParams = {}) =>
    this.request<WormholeQueryGetConfigResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/config`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryConsensusGuardianSetIndex
   * @summary Queries a ConsensusGuardianSetIndex by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/consensus_guardian_set_index
   */
  queryConsensusGuardianSetIndex = (params: RequestParams = {}) =>
    this.request<WormholeQueryGetConsensusGuardianSetIndexResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/consensus_guardian_set_index`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGuardianSetAll
   * @summary Queries a list of guardianSet items.
   * @request GET:/wormhole_foundation/wormchain/wormhole/guardianSet
   */
  queryGuardianSetAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllGuardianSetResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/guardianSet`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGuardianSet
   * @summary Queries a guardianSet by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/guardianSet/{index}
   */
  queryGuardianSet = (index: number, params: RequestParams = {}) =>
    this.request<WormholeQueryGetGuardianSetResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/guardianSet/${index}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGuardianValidatorAll
   * @summary Queries a list of GuardianValidator items.
   * @request GET:/wormhole_foundation/wormchain/wormhole/guardian_validator
   */
  queryGuardianValidatorAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllGuardianValidatorResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/guardian_validator`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGuardianValidator
   * @summary Queries a GuardianValidator by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/guardian_validator/{guardianKey}
   */
  queryGuardianValidator = (guardianKey: string, params: RequestParams = {}) =>
    this.request<WormholeQueryGetGuardianValidatorResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/guardian_validator/${guardianKey}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryIbcComposabilityMwContract
   * @request GET:/wormhole_foundation/wormchain/wormhole/ibc_composability_mw_contract
   */
  queryIbcComposabilityMwContract = (params: RequestParams = {}) =>
    this.request<WormholeQueryIbcComposabilityMwContractResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/ibc_composability_mw_contract`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryLatestGuardianSetIndex
   * @summary Queries a list of LatestGuardianSetIndex items.
   * @request GET:/wormhole_foundation/wormchain/wormhole/latest_guardian_set_index
   */
  queryLatestGuardianSetIndex = (params: RequestParams = {}) =>
    this.request<WormholeQueryLatestGuardianSetIndexResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/latest_guardian_set_index`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryReplayProtectionAll
   * @summary Queries a list of replayProtection items.
   * @request GET:/wormhole_foundation/wormchain/wormhole/replayProtection
   */
  queryReplayProtectionAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllReplayProtectionResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/replayProtection`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryReplayProtection
   * @summary Queries a replayProtection by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/replayProtection/{index}
   */
  queryReplayProtection = (index: string, params: RequestParams = {}) =>
    this.request<WormholeQueryGetReplayProtectionResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/replayProtection/${index}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QuerySequenceCounterAll
   * @summary Queries a list of sequenceCounter items.
   * @request GET:/wormhole_foundation/wormchain/wormhole/sequenceCounter
   */
  querySequenceCounterAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllSequenceCounterResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/sequenceCounter`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QuerySequenceCounter
   * @summary Queries a sequenceCounter by index.
   * @request GET:/wormhole_foundation/wormchain/wormhole/sequenceCounter/{index}
   */
  querySequenceCounter = (index: string, params: RequestParams = {}) =>
    this.request<WormholeQueryGetSequenceCounterResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/sequenceCounter/${index}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryWasmInstantiateAllowlistAll
   * @request GET:/wormhole_foundation/wormchain/wormhole/wasm_instantiate_allowlist
   */
  queryWasmInstantiateAllowlistAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<WormholeQueryAllWasmInstantiateAllowlistResponse, RpcStatus>({
      path: `/wormhole_foundation/wormchain/wormhole/wasm_instantiate_allowlist`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });
}
