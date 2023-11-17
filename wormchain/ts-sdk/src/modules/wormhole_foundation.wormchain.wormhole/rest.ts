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
  /** @format byte */
  next_key?: string;

  /** @format uint64 */
  total?: string;
}

export interface WormholeConfig {
  /** @format uint64 */
  guardian_set_expiration?: string;

  /** @format byte */
  governance_emitter?: string;

  /** @format int64 */
  governance_chain?: number;

  /** @format int64 */
  chain_id?: number;
}

export interface WormholeConsensusGuardianSetIndex {
  /** @format int64 */
  index?: number;
}

export type WormholeEmptyResponse = object;

export interface WormholeGuardianSet {
  /** @format int64 */
  index?: number;
  keys?: string[];

  /** @format uint64 */
  expirationTime?: string;
}

export interface WormholeGuardianValidator {
  /** @format byte */
  guardianKey?: string;

  /** @format byte */
  validatorAddr?: string;
}

export type WormholeMsgAllowlistResponse = object;

export type WormholeMsgExecuteGovernanceVAAResponse = object;

export interface WormholeMsgInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address?: string;

  /** @format byte */
  data?: string;
}

/**
 * MsgMigrateContractResponse returns contract migration result data.
 */
export interface WormholeMsgMigrateContractResponse {
  /** @format byte */
  data?: string;
}

export type WormholeMsgRegisterAccountAsGuardianResponse = object;

export interface WormholeMsgStoreCodeResponse {
  /** @format uint64 */
  code_id?: string;

  /** @format byte */
  checksum?: string;
}

export type WormholeMsgWasmInstantiateAllowlistResponse = object;

export interface WormholeQueryAllGuardianSetResponse {
  GuardianSet?: WormholeGuardianSet[];

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
  guardianValidator?: WormholeGuardianValidator[];

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
  replayProtection?: WormholeReplayProtection[];

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
  sequenceCounter?: WormholeSequenceCounter[];

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
  Config?: WormholeConfig;
}

export interface WormholeQueryGetConsensusGuardianSetIndexResponse {
  ConsensusGuardianSetIndex?: WormholeConsensusGuardianSetIndex;
}

export interface WormholeQueryGetGuardianSetResponse {
  GuardianSet?: WormholeGuardianSet;
}

export interface WormholeQueryGetGuardianValidatorResponse {
  guardianValidator?: WormholeGuardianValidator;
}

export interface WormholeQueryGetReplayProtectionResponse {
  replayProtection?: WormholeReplayProtection;
}

export interface WormholeQueryGetSequenceCounterResponse {
  sequenceCounter?: WormholeSequenceCounter;
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

export interface WormholeReplayProtection {
  index?: string;
}

export interface WormholeSequenceCounter {
  index?: string;

  /** @format uint64 */
  sequence?: string;
}

export interface WormholeValidatorAllowedAddress {
  validator_address?: string;
  allowed_address?: string;
  name?: string;
}

export interface WormholeWasmInstantiateAllowedContractCodeId {
  contract_address?: string;

  /** @format uint64 */
  code_id?: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: keyof Omit<Body, "body" | "bodyUsed">;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<FullRequestParams, "body" | "method" | "query" | "path">;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (securityData: SecurityDataType) => RequestParams | void;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown> extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "";
  private securityData: SecurityDataType = null as any;
  private securityWorker: null | ApiConfig<SecurityDataType>["securityWorker"] = null;
  private abortControllers = new Map<CancelToken, AbortController>();

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType) => {
    this.securityData = data;
  };

  private addQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];

    return (
      encodeURIComponent(key) +
      "=" +
      encodeURIComponent(Array.isArray(value) ? value.join(",") : typeof value === "number" ? value : `${value}`)
    );
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter((key) => "undefined" !== typeof query[key]);
    return keys
      .map((key) =>
        typeof query[key] === "object" && !Array.isArray(query[key])
          ? this.toQueryString(query[key] as QueryParamsType)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string") ? JSON.stringify(input) : input,
    [ContentType.FormData]: (input: any) =>
      Object.keys(input || {}).reduce((data, key) => {
        data.append(key, input[key]);
        return data;
      }, new FormData()),
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  private mergeRequestParams(params1: RequestParams, params2?: RequestParams): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  private createAbortSignal = (cancelToken: CancelToken): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format = "json",
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams = (secure && this.securityWorker && this.securityWorker(this.securityData)) || {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];

    return fetch(`${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`, {
      ...requestParams,
      headers: {
        ...(type && type !== ContentType.FormData ? { "Content-Type": type } : {}),
        ...(requestParams.headers || {}),
      },
      signal: cancelToken ? this.createAbortSignal(cancelToken) : void 0,
      body: typeof body === "undefined" || body === null ? null : payloadFormatter(body),
    }).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = (null as unknown) as T;
      r.error = (null as unknown) as E;

      const data = await response[format]()
        .then((data) => {
          if (r.ok) {
            r.data = data;
          } else {
            r.error = data;
          }
          return r;
        })
        .catch((e) => {
          r.error = e;
          return r;
        });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title wormhole/config.proto
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
    validator_address: string,
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
      path: `/wormhole_foundation/wormchain/wormhole/allowlist/${validator_address}`,
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
