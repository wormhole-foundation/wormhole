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
* `Any` contains an arbitrary serialized protocol buffer message along with a
URL that describes the type of the serialized message.

Protobuf library provides support to pack/unpack Any values in the form
of utility functions or additional generated methods of the Any type.

Example 1: Pack and unpack a message in C++.

    Foo foo = ...;
    Any any;
    any.PackFrom(foo);
    ...
    if (any.UnpackTo(&foo)) {
      ...
    }

Example 2: Pack and unpack a message in Java.

    Foo foo = ...;
    Any any = Any.pack(foo);
    ...
    if (any.is(Foo.class)) {
      foo = any.unpack(Foo.class);
    }

 Example 3: Pack and unpack a message in Python.

    foo = Foo(...)
    any = Any()
    any.Pack(foo)
    ...
    if any.Is(Foo.DESCRIPTOR):
      any.Unpack(foo)
      ...

 Example 4: Pack and unpack a message in Go

     foo := &pb.Foo{...}
     any, err := anypb.New(foo)
     if err != nil {
       ...
     }
     ...
     foo := &pb.Foo{}
     if err := any.UnmarshalTo(foo); err != nil {
       ...
     }

The pack methods provided by protobuf library will by default use
'type.googleapis.com/full.type.name' as the type URL and the unpack
methods only use the fully qualified type name after the last '/'
in the type URL, for example "foo.bar.com/x/y.z" will yield type
name "y.z".


JSON
====
The JSON representation of an `Any` value uses the regular
representation of the deserialized, embedded message, with an
additional field `@type` which contains the type URL. Example:

    package google.profile;
    message Person {
      string first_name = 1;
      string last_name = 2;
    }

    {
      "@type": "type.googleapis.com/google.profile.Person",
      "firstName": <string>,
      "lastName": <string>
    }

If the embedded message type is well-known and has a custom JSON
representation, that representation will be embedded adding a field
`value` which holds the custom JSON in addition to the `@type`
field. Example (for message [google.protobuf.Duration][]):

    {
      "@type": "type.googleapis.com/google.protobuf.Duration",
      "value": "1.212s"
    }
*/
export interface ProtobufAny {
  /**
   * A URL/resource name that uniquely identifies the type of the serialized
   * protocol buffer message. This string must contain at least
   * one "/" character. The last segment of the URL's path must represent
   * the fully qualified name of the type (as in
   * `path/google.protobuf.Duration`). The name should be in a canonical form
   * (e.g., leading "." is not accepted).
   *
   * In practice, teams usually precompile into the binary all types that they
   * expect it to use in the context of Any. However, for URLs which use the
   * scheme `http`, `https`, or no scheme, one can optionally set up a type
   * server that maps type URLs to message definitions as follows:
   * * If no scheme is provided, `https` is assumed.
   * * An HTTP GET on the URL must yield a [google.protobuf.Type][]
   *   value in binary format, or produce an error.
   * * Applications are allowed to cache lookup results based on the
   *   URL, or have them precompiled into a binary to avoid any
   *   lookup. Therefore, binary compatibility needs to be preserved
   *   on changes to types. (Use versioned type names to manage
   *   breaking changes.)
   * Note: this functionality is not currently available in the official
   * protobuf release, and it is not used for type URLs beginning with
   * type.googleapis.com.
   * Schemes other than `http`, `https` (or the empty scheme) might be
   * used with implementation specific semantics.
   */
  "@type"?: string;
}

export interface RpcStatus {
  /** @format int32 */
  code?: number;
  message?: string;
  details?: ProtobufAny[];
}

/**
* AbsoluteTxPosition is a unique transaction position that allows for global
ordering of transactions.
*/
export interface V1AbsoluteTxPosition {
  /**
   * BlockHeight is the block the contract was created at
   * @format uint64
   */
  block_height?: string;

  /**
   * TxIndex is a monotonic counter within the block (actual transaction index,
   * or gas consumed)
   * @format uint64
   */
  tx_index?: string;
}

/**
 * AccessConfig access control type.
 */
export interface V1AccessConfig {
  /**
   * - ACCESS_TYPE_UNSPECIFIED: AccessTypeUnspecified placeholder for empty value
   *  - ACCESS_TYPE_NOBODY: AccessTypeNobody forbidden
   *  - ACCESS_TYPE_EVERYBODY: AccessTypeEverybody unrestricted
   *  - ACCESS_TYPE_ANY_OF_ADDRESSES: AccessTypeAnyOfAddresses allow any of the addresses
   */
  permission?: V1AccessType;
  addresses?: string[];
}

/**
* - ACCESS_TYPE_UNSPECIFIED: AccessTypeUnspecified placeholder for empty value
 - ACCESS_TYPE_NOBODY: AccessTypeNobody forbidden
 - ACCESS_TYPE_EVERYBODY: AccessTypeEverybody unrestricted
 - ACCESS_TYPE_ANY_OF_ADDRESSES: AccessTypeAnyOfAddresses allow any of the addresses
*/
export enum V1AccessType {
  ACCESS_TYPE_UNSPECIFIED = "ACCESS_TYPE_UNSPECIFIED",
  ACCESS_TYPE_NOBODY = "ACCESS_TYPE_NOBODY",
  ACCESS_TYPE_EVERYBODY = "ACCESS_TYPE_EVERYBODY",
  ACCESS_TYPE_ANY_OF_ADDRESSES = "ACCESS_TYPE_ANY_OF_ADDRESSES",
}

export interface V1CodeInfoResponse {
  /**
   * id for legacy support
   * @format uint64
   */
  code_id?: string;
  creator?: string;

  /** @format byte */
  data_hash?: string;

  /** AccessConfig access control type. */
  instantiate_permission?: V1AccessConfig;
}

/**
 * ContractCodeHistoryEntry metadata to a contract.
 */
export interface V1ContractCodeHistoryEntry {
  /**
   * - CONTRACT_CODE_HISTORY_OPERATION_TYPE_UNSPECIFIED: ContractCodeHistoryOperationTypeUnspecified placeholder for empty value
   *  - CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT: ContractCodeHistoryOperationTypeInit on chain contract instantiation
   *  - CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE: ContractCodeHistoryOperationTypeMigrate code migration
   *  - CONTRACT_CODE_HISTORY_OPERATION_TYPE_GENESIS: ContractCodeHistoryOperationTypeGenesis based on genesis data
   */
  operation?: V1ContractCodeHistoryOperationType;

  /**
   * CodeID is the reference to the stored WASM code
   * @format uint64
   */
  code_id?: string;

  /** Updated Tx position when the operation was executed. */
  updated?: V1AbsoluteTxPosition;

  /** @format byte */
  msg?: string;
}

/**
* - CONTRACT_CODE_HISTORY_OPERATION_TYPE_UNSPECIFIED: ContractCodeHistoryOperationTypeUnspecified placeholder for empty value
 - CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT: ContractCodeHistoryOperationTypeInit on chain contract instantiation
 - CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE: ContractCodeHistoryOperationTypeMigrate code migration
 - CONTRACT_CODE_HISTORY_OPERATION_TYPE_GENESIS: ContractCodeHistoryOperationTypeGenesis based on genesis data
*/
export enum V1ContractCodeHistoryOperationType {
  CONTRACT_CODE_HISTORY_OPERATION_TYPE_UNSPECIFIED = "CONTRACT_CODE_HISTORY_OPERATION_TYPE_UNSPECIFIED",
  CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT = "CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT",
  CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE = "CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE",
  CONTRACT_CODE_HISTORY_OPERATION_TYPE_GENESIS = "CONTRACT_CODE_HISTORY_OPERATION_TYPE_GENESIS",
}

export interface V1Model {
  /**
   * hex-encode key to read it better (this is often ascii)
   * @format byte
   */
  key?: string;

  /**
   * base64-encode raw value
   * @format byte
   */
  value?: string;
}

/**
* MsgAddCodeUploadParamsAddressesResponse defines the response
structure for executing a MsgAddCodeUploadParamsAddresses message.
*/
export type V1MsgAddCodeUploadParamsAddressesResponse = object;

export type V1MsgClearAdminResponse = object;

/**
 * MsgExecuteContractResponse returns execution result data.
 */
export interface V1MsgExecuteContractResponse {
  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

export interface V1MsgInstantiateContract2Response {
  /** Address is the bech32 address of the new contract instance. */
  address?: string;

  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

export interface V1MsgInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address?: string;

  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

/**
 * MsgMigrateContractResponse returns contract migration result data.
 */
export interface V1MsgMigrateContractResponse {
  /**
   * Data contains same raw bytes returned as data from the wasm contract.
   * (May be empty)
   * @format byte
   */
  data?: string;
}

/**
* MsgPinCodesResponse defines the response structure for executing a
MsgPinCodes message.

Since: 0.40
*/
export type V1MsgPinCodesResponse = object;

/**
* MsgRemoveCodeUploadParamsAddressesResponse defines the response
structure for executing a MsgRemoveCodeUploadParamsAddresses message.
*/
export type V1MsgRemoveCodeUploadParamsAddressesResponse = object;

/**
* MsgStoreAndInstantiateContractResponse defines the response structure
for executing a MsgStoreAndInstantiateContract message.

Since: 0.40
*/
export interface V1MsgStoreAndInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address?: string;

  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

/**
* MsgStoreAndMigrateContractResponse defines the response structure
for executing a MsgStoreAndMigrateContract message.

Since: 0.42
*/
export interface V1MsgStoreAndMigrateContractResponse {
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

  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

/**
 * MsgStoreCodeResponse returns store result data.
 */
export interface V1MsgStoreCodeResponse {
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

/**
* MsgSudoContractResponse defines the response structure for executing a
MsgSudoContract message.

Since: 0.40
*/
export interface V1MsgSudoContractResponse {
  /**
   * Data contains bytes to returned from the contract
   * @format byte
   */
  data?: string;
}

/**
* MsgUnpinCodesResponse defines the response structure for executing a
MsgUnpinCodes message.

Since: 0.40
*/
export type V1MsgUnpinCodesResponse = object;

export type V1MsgUpdateAdminResponse = object;

export type V1MsgUpdateContractLabelResponse = object;

export type V1MsgUpdateInstantiateConfigResponse = object;

/**
* MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.

Since: 0.40
*/
export type V1MsgUpdateParamsResponse = object;

export interface V1QueryAllContractStateResponse {
  models?: V1Model[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

export interface V1QueryCodeResponse {
  code_info?: V1CodeInfoResponse;

  /** @format byte */
  data?: string;
}

export interface V1QueryCodesResponse {
  code_infos?: V1CodeInfoResponse[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

export interface V1QueryContractHistoryResponse {
  entries?: V1ContractCodeHistoryEntry[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

export interface V1QueryContractInfoResponse {
  /** address is the address of the contract */
  address?: string;
  contract_info?: Wasmv1ContractInfo;
}

export interface V1QueryContractsByCodeResponse {
  /** contracts are a set of contract addresses */
  contracts?: string[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
* QueryContractsByCreatorResponse is the response type for the
Query/ContractsByCreator RPC method.
*/
export interface V1QueryContractsByCreatorResponse {
  /** ContractAddresses result set */
  contract_addresses?: string[];

  /** Pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * QueryParamsResponse is the response type for the Query/Params RPC method.
 */
export interface V1QueryParamsResponse {
  /** params defines the parameters of the module. */
  params?: Wasmv1Params;
}

export interface V1QueryPinnedCodesResponse {
  code_ids?: string[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

export interface V1QueryRawContractStateResponse {
  /**
   * Data contains the raw store data
   * @format byte
   */
  data?: string;
}

export interface V1QuerySmartContractStateResponse {
  /**
   * Data contains the json data returned from the smart contract
   * @format byte
   */
  data?: string;
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

export interface Wasmv1ContractInfo {
  /**
   * CodeID is the reference to the stored Wasm code
   * @format uint64
   */
  code_id?: string;

  /** Creator address who initially instantiated the contract */
  creator?: string;

  /** Admin is an optional address that can execute migrations */
  admin?: string;

  /** Label is optional metadata to be stored with a contract instance. */
  label?: string;

  /** Created Tx position when the contract was instantiated. */
  created?: V1AbsoluteTxPosition;
  ibc_port_id?: string;

  /**
   * Extension is an extension point to store custom metadata within the
   * persistence model.
   */
  extension?: ProtobufAny;
}

/**
 * Params defines the set of wasm parameters.
 */
export interface Wasmv1Params {
  /** AccessConfig access control type. */
  code_upload_access?: V1AccessConfig;

  /**
   * - ACCESS_TYPE_UNSPECIFIED: AccessTypeUnspecified placeholder for empty value
   *  - ACCESS_TYPE_NOBODY: AccessTypeNobody forbidden
   *  - ACCESS_TYPE_EVERYBODY: AccessTypeEverybody unrestricted
   *  - ACCESS_TYPE_ANY_OF_ADDRESSES: AccessTypeAnyOfAddresses allow any of the addresses
   */
  instantiate_default_permission?: V1AccessType;
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
 * @title cosmwasm/wasm/v1/authz.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryCodes
   * @summary Codes gets the metadata for all stored wasm codes
   * @request GET:/cosmwasm/wasm/v1/code
   */
  queryCodes = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryCodesResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/code`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryCode
   * @summary Code gets the binary code and metadata for a singe wasm code
   * @request GET:/cosmwasm/wasm/v1/code/{code_id}
   */
  queryCode = (codeId: string, params: RequestParams = {}) =>
    this.request<V1QueryCodeResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/code/${codeId}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryContractsByCode
   * @summary ContractsByCode lists all smart contracts for a code id
   * @request GET:/cosmwasm/wasm/v1/code/{code_id}/contracts
   */
  queryContractsByCode = (
    codeId: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryContractsByCodeResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/code/${codeId}/contracts`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryParams
   * @summary Params gets the module params
   * @request GET:/cosmwasm/wasm/v1/codes/params
   */
  queryParams = (params: RequestParams = {}) =>
    this.request<V1QueryParamsResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/codes/params`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryPinnedCodes
   * @summary PinnedCodes gets the pinned code ids
   * @request GET:/cosmwasm/wasm/v1/codes/pinned
   */
  queryPinnedCodes = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryPinnedCodesResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/codes/pinned`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryContractInfo
   * @summary ContractInfo gets the contract meta data
   * @request GET:/cosmwasm/wasm/v1/contract/{address}
   */
  queryContractInfo = (address: string, params: RequestParams = {}) =>
    this.request<V1QueryContractInfoResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contract/${address}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryContractHistory
   * @summary ContractHistory gets the contract code history
   * @request GET:/cosmwasm/wasm/v1/contract/{address}/history
   */
  queryContractHistory = (
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
    this.request<V1QueryContractHistoryResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contract/${address}/history`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryRawContractState
   * @summary RawContractState gets single key from the raw store data of a contract
   * @request GET:/cosmwasm/wasm/v1/contract/{address}/raw/{query_data}
   */
  queryRawContractState = (address: string, queryData: string, params: RequestParams = {}) =>
    this.request<V1QueryRawContractStateResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contract/${address}/raw/${queryData}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QuerySmartContractState
   * @summary SmartContractState get smart query result from the contract
   * @request GET:/cosmwasm/wasm/v1/contract/{address}/smart/{query_data}
   */
  querySmartContractState = (address: string, queryData: string, params: RequestParams = {}) =>
    this.request<V1QuerySmartContractStateResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contract/${address}/smart/${queryData}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryAllContractState
   * @summary AllContractState gets all raw store data for a single contract
   * @request GET:/cosmwasm/wasm/v1/contract/{address}/state
   */
  queryAllContractState = (
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
    this.request<V1QueryAllContractStateResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contract/${address}/state`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryContractsByCreator
   * @summary ContractsByCreator gets the contracts by creator
   * @request GET:/cosmwasm/wasm/v1/contracts/creator/{creator_address}
   */
  queryContractsByCreator = (
    creatorAddress: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryContractsByCreatorResponse, RpcStatus>({
      path: `/cosmwasm/wasm/v1/contracts/creator/${creatorAddress}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });
}
