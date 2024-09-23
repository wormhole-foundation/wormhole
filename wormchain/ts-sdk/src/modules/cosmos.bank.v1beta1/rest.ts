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

/**
 * Params defines the parameters for the bank module.
 */
export interface Bankv1Beta1Params {
  /**
   * Deprecated: Use of SendEnabled in params is deprecated.
   * For genesis, use the newly added send_enabled field in the genesis object.
   * Storage, lookup, and manipulation of this information is now in the keeper.
   *
   * As of cosmos-sdk 0.47, this only exists for backwards compatibility of genesis files.
   */
  send_enabled?: Bankv1Beta1SendEnabled[];
  default_send_enabled?: boolean;
}

/**
* SendEnabled maps coin denom to a send_enabled status (whether a denom is
sendable).
*/
export interface Bankv1Beta1SendEnabled {
  denom?: string;
  enabled?: boolean;
}

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
* Coin defines a token with a denomination and an amount.

NOTE: The amount field is an Int which implements the custom method
signatures required by gogoproto.
*/
export interface V1Beta1Coin {
  denom?: string;
  amount?: string;
}

/**
* DenomOwner defines structure representing an account that owns or holds a
particular denominated token. It contains the account address and account
balance of the denominated token.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1DenomOwner {
  /** address defines the address that owns a particular denomination. */
  address?: string;

  /** balance is the balance of the denominated coin for an account. */
  balance?: V1Beta1Coin;
}

/**
* DenomUnit represents a struct that describes a given
denomination unit of the basic token.
*/
export interface V1Beta1DenomUnit {
  /** denom represents the string name of the given denom unit (e.g uatom). */
  denom?: string;

  /**
   * exponent represents power of 10 exponent that one must
   * raise the base_denom to in order to equal the given DenomUnit's denom
   * 1 denom = 10^exponent base_denom
   * (e.g. with a base_denom of uatom, one can create a DenomUnit of 'atom' with
   * exponent = 6, thus: 1 atom = 10^6 uatom).
   * @format int64
   */
  exponent?: number;

  /** aliases is a list of string aliases for the given denom */
  aliases?: string[];
}

/**
 * Input models transaction input.
 */
export interface V1Beta1Input {
  address?: string;
  coins?: V1Beta1Coin[];
}

/**
* Metadata represents a struct that describes
a basic token.
*/
export interface V1Beta1Metadata {
  description?: string;

  /** denom_units represents the list of DenomUnit's for a given coin */
  denom_units?: V1Beta1DenomUnit[];

  /** base represents the base denom (should be the DenomUnit with exponent = 0). */
  base?: string;

  /**
   * display indicates the suggested denom that should be
   * displayed in clients.
   */
  display?: string;

  /**
   * name defines the name of the token (eg: Cosmos Atom)
   * Since: cosmos-sdk 0.43
   */
  name?: string;

  /**
   * symbol is the token symbol usually shown on exchanges (eg: ATOM). This can
   * be the same as the display.
   *
   * Since: cosmos-sdk 0.43
   */
  symbol?: string;

  /**
   * URI to a document (on or off-chain) that contains additional information. Optional.
   *
   * Since: cosmos-sdk 0.46
   */
  uri?: string;

  /**
   * URIHash is a sha256 hash of a document pointed by URI. It's used to verify that
   * the document didn't change. Optional.
   *
   * Since: cosmos-sdk 0.46
   */
  uri_hash?: string;
}

/**
 * MsgMultiSendResponse defines the Msg/MultiSend response type.
 */
export type V1Beta1MsgMultiSendResponse = object;

/**
 * MsgSendResponse defines the Msg/Send response type.
 */
export type V1Beta1MsgSendResponse = object;

/**
* MsgSetSendEnabledResponse defines the Msg/SetSendEnabled response type.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgSetSendEnabledResponse = object;

/**
* MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgUpdateParamsResponse = object;

/**
 * Output models transaction outputs.
 */
export interface V1Beta1Output {
  address?: string;
  coins?: V1Beta1Coin[];
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

/**
* QueryAllBalancesResponse is the response type for the Query/AllBalances RPC
method.
*/
export interface V1Beta1QueryAllBalancesResponse {
  /** balances is the balances of all the coins. */
  balances?: V1Beta1Coin[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * QueryBalanceResponse is the response type for the Query/Balance RPC method.
 */
export interface V1Beta1QueryBalanceResponse {
  /** balance is the balance of the coin. */
  balance?: V1Beta1Coin;
}

/**
* QueryDenomMetadataResponse is the response type for the Query/DenomMetadata RPC
method.
*/
export interface V1Beta1QueryDenomMetadataResponse {
  /** metadata describes and provides all the client information for the requested token. */
  metadata?: V1Beta1Metadata;
}

/**
* QueryDenomOwnersResponse defines the RPC response of a DenomOwners RPC query.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1QueryDenomOwnersResponse {
  denom_owners?: V1Beta1DenomOwner[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
* QueryDenomsMetadataResponse is the response type for the Query/DenomsMetadata RPC
method.
*/
export interface V1Beta1QueryDenomsMetadataResponse {
  /** metadata provides the client information for all the registered tokens. */
  metadatas?: V1Beta1Metadata[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * QueryParamsResponse defines the response type for querying x/bank parameters.
 */
export interface V1Beta1QueryParamsResponse {
  /** Params defines the parameters for the bank module. */
  params?: Bankv1Beta1Params;
}

/**
* QuerySendEnabledResponse defines the RPC response of a SendEnable query.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1QuerySendEnabledResponse {
  send_enabled?: Bankv1Beta1SendEnabled[];

  /**
   * pagination defines the pagination in the response. This field is only
   * populated if the denoms field in the request is empty.
   */
  pagination?: V1Beta1PageResponse;
}

/**
* QuerySpendableBalanceByDenomResponse defines the gRPC response structure for
querying an account's spendable balance for a specific denom.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1QuerySpendableBalanceByDenomResponse {
  /** balance is the balance of the coin. */
  balance?: V1Beta1Coin;
}

/**
* QuerySpendableBalancesResponse defines the gRPC response structure for querying
an account's spendable balances.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1QuerySpendableBalancesResponse {
  /** balances is the spendable balances of all the coins. */
  balances?: V1Beta1Coin[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * QuerySupplyOfResponse is the response type for the Query/SupplyOf RPC method.
 */
export interface V1Beta1QuerySupplyOfResponse {
  /** amount is the supply of the coin. */
  amount?: V1Beta1Coin;
}

export interface V1Beta1QueryTotalSupplyResponse {
  /** supply is the supply of the coins */
  supply?: V1Beta1Coin[];

  /**
   * pagination defines the pagination in the response.
   *
   * Since: cosmos-sdk 0.43
   */
  pagination?: V1Beta1PageResponse;
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
 * @title cosmos/bank/v1beta1/authz.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set.
   *
   * @tags Query
   * @name QueryAllBalances
   * @summary AllBalances queries the balance of all coins for a single account.
   * @request GET:/cosmos/bank/v1beta1/balances/{address}
   */
  queryAllBalances = (
    address: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryAllBalancesResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/balances/${address}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryBalance
   * @summary Balance queries the balance of a single coin for a single account.
   * @request GET:/cosmos/bank/v1beta1/balances/{address}/by_denom
   */
  queryBalance = (address: string, query?: { denom?: string }, params: RequestParams = {}) =>
    this.request<V1Beta1QueryBalanceResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/balances/${address}/by_denom`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set. Since: cosmos-sdk 0.46
 * 
 * @tags Query
 * @name QueryDenomOwners
 * @summary DenomOwners queries for all account addresses that own a particular token
denomination.
 * @request GET:/cosmos/bank/v1beta1/denom_owners/{denom}
 */
  queryDenomOwners = (
    denom: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryDenomOwnersResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/denom_owners/${denom}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryDenomsMetadata
 * @summary DenomsMetadata queries the client metadata for all registered coin
denominations.
 * @request GET:/cosmos/bank/v1beta1/denoms_metadata
 */
  queryDenomsMetadata = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryDenomsMetadataResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/denoms_metadata`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryDenomMetadata
   * @summary DenomsMetadata queries the client metadata of a given coin denomination.
   * @request GET:/cosmos/bank/v1beta1/denoms_metadata/{denom}
   */
  queryDenomMetadata = (denom: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDenomMetadataResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/denoms_metadata/${denom}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryParams
   * @summary Params queries the parameters of x/bank module.
   * @request GET:/cosmos/bank/v1beta1/params
   */
  queryParams = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryParamsResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/params`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description This query only returns denominations that have specific SendEnabled settings. Any denomination that does not have a specific setting will use the default params.default_send_enabled, and will not be returned by this query. Since: cosmos-sdk 0.47
   *
   * @tags Query
   * @name QuerySendEnabled
   * @summary SendEnabled queries for SendEnabled entries.
   * @request GET:/cosmos/bank/v1beta1/send_enabled
   */
  querySendEnabled = (
    query?: {
      denoms?: string[];
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QuerySendEnabledResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/send_enabled`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set. Since: cosmos-sdk 0.46
 * 
 * @tags Query
 * @name QuerySpendableBalances
 * @summary SpendableBalances queries the spendable balance of all coins for a single
account.
 * @request GET:/cosmos/bank/v1beta1/spendable_balances/{address}
 */
  querySpendableBalances = (
    address: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QuerySpendableBalancesResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/spendable_balances/${address}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set. Since: cosmos-sdk 0.47
 * 
 * @tags Query
 * @name QuerySpendableBalanceByDenom
 * @summary SpendableBalanceByDenom queries the spendable balance of a single denom for
a single account.
 * @request GET:/cosmos/bank/v1beta1/spendable_balances/{address}/by_denom
 */
  querySpendableBalanceByDenom = (address: string, query?: { denom?: string }, params: RequestParams = {}) =>
    this.request<V1Beta1QuerySpendableBalanceByDenomResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/spendable_balances/${address}/by_denom`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set.
   *
   * @tags Query
   * @name QueryTotalSupply
   * @summary TotalSupply queries the total supply of all coins.
   * @request GET:/cosmos/bank/v1beta1/supply
   */
  queryTotalSupply = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryTotalSupplyResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/supply`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set.
   *
   * @tags Query
   * @name QuerySupplyOf
   * @summary SupplyOf queries the supply of a single coin.
   * @request GET:/cosmos/bank/v1beta1/supply/by_denom
   */
  querySupplyOf = (query?: { denom?: string }, params: RequestParams = {}) =>
    this.request<V1Beta1QuerySupplyOfResponse, RpcStatus>({
      path: `/cosmos/bank/v1beta1/supply/by_denom`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });
}
