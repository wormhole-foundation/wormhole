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
 * Params defines the parameters for the auth module.
 */
export interface Authv1Beta1Params {
  /** @format uint64 */
  max_memo_characters?: string;

  /** @format uint64 */
  tx_sig_limit?: string;

  /** @format uint64 */
  tx_size_cost_per_byte?: string;

  /** @format uint64 */
  sig_verify_cost_ed25519?: string;

  /** @format uint64 */
  sig_verify_cost_secp256k1?: string;
}

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
* AddressBytesToStringResponse is the response type for AddressString rpc method.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1AddressBytesToStringResponse {
  address_string?: string;
}

/**
* AddressStringToBytesResponse is the response type for AddressBytes rpc method.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1AddressStringToBytesResponse {
  /** @format byte */
  address_bytes?: string;
}

/**
* BaseAccount defines a base account type. It contains all the necessary fields
for basic account functionality. Any custom account type should extend this
type for additional functionality (e.g. vesting).
*/
export interface V1Beta1BaseAccount {
  address?: string;

  /**
   * `Any` contains an arbitrary serialized protocol buffer message along with a
   * URL that describes the type of the serialized message.
   *
   * Protobuf library provides support to pack/unpack Any values in the form
   * of utility functions or additional generated methods of the Any type.
   * Example 1: Pack and unpack a message in C++.
   *     Foo foo = ...;
   *     Any any;
   *     any.PackFrom(foo);
   *     ...
   *     if (any.UnpackTo(&foo)) {
   *       ...
   *     }
   * Example 2: Pack and unpack a message in Java.
   *     Any any = Any.pack(foo);
   *     if (any.is(Foo.class)) {
   *       foo = any.unpack(Foo.class);
   *  Example 3: Pack and unpack a message in Python.
   *     foo = Foo(...)
   *     any = Any()
   *     any.Pack(foo)
   *     if any.Is(Foo.DESCRIPTOR):
   *       any.Unpack(foo)
   *  Example 4: Pack and unpack a message in Go
   *      foo := &pb.Foo{...}
   *      any, err := anypb.New(foo)
   *      if err != nil {
   *        ...
   *      }
   *      ...
   *      foo := &pb.Foo{}
   *      if err := any.UnmarshalTo(foo); err != nil {
   * The pack methods provided by protobuf library will by default use
   * 'type.googleapis.com/full.type.name' as the type URL and the unpack
   * methods only use the fully qualified type name after the last '/'
   * in the type URL, for example "foo.bar.com/x/y.z" will yield type
   * name "y.z".
   * JSON
   * ====
   * The JSON representation of an `Any` value uses the regular
   * representation of the deserialized, embedded message, with an
   * additional field `@type` which contains the type URL. Example:
   *     package google.profile;
   *     message Person {
   *       string first_name = 1;
   *       string last_name = 2;
   *     {
   *       "@type": "type.googleapis.com/google.profile.Person",
   *       "firstName": <string>,
   *       "lastName": <string>
   * If the embedded message type is well-known and has a custom JSON
   * representation, that representation will be embedded adding a field
   * `value` which holds the custom JSON in addition to the `@type`
   * field. Example (for message [google.protobuf.Duration][]):
   *       "@type": "type.googleapis.com/google.protobuf.Duration",
   *       "value": "1.212s"
   */
  pub_key?: ProtobufAny;

  /** @format uint64 */
  account_number?: string;

  /** @format uint64 */
  sequence?: string;
}

/**
* Bech32PrefixResponse is the response type for Bech32Prefix rpc method.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1Bech32PrefixResponse {
  bech32_prefix?: string;
}

/**
* MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgUpdateParamsResponse = object;

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
 * Since: cosmos-sdk 0.46.2
 */
export interface V1Beta1QueryAccountAddressByIDResponse {
  account_address?: string;
}

/**
* QueryAccountInfoResponse is the Query/AccountInfo response type.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1QueryAccountInfoResponse {
  /** info is the account info which is represented by BaseAccount. */
  info?: V1Beta1BaseAccount;
}

/**
 * QueryAccountResponse is the response type for the Query/Account RPC method.
 */
export interface V1Beta1QueryAccountResponse {
  /** account defines the account of the corresponding address. */
  account?: ProtobufAny;
}

/**
* QueryAccountsResponse is the response type for the Query/Accounts RPC method.

Since: cosmos-sdk 0.43
*/
export interface V1Beta1QueryAccountsResponse {
  /** accounts are the existing accounts */
  accounts?: ProtobufAny[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * QueryModuleAccountByNameResponse is the response type for the Query/ModuleAccountByName RPC method.
 */
export interface V1Beta1QueryModuleAccountByNameResponse {
  /**
   * `Any` contains an arbitrary serialized protocol buffer message along with a
   * URL that describes the type of the serialized message.
   *
   * Protobuf library provides support to pack/unpack Any values in the form
   * of utility functions or additional generated methods of the Any type.
   * Example 1: Pack and unpack a message in C++.
   *     Foo foo = ...;
   *     Any any;
   *     any.PackFrom(foo);
   *     ...
   *     if (any.UnpackTo(&foo)) {
   *       ...
   *     }
   * Example 2: Pack and unpack a message in Java.
   *     Any any = Any.pack(foo);
   *     if (any.is(Foo.class)) {
   *       foo = any.unpack(Foo.class);
   *  Example 3: Pack and unpack a message in Python.
   *     foo = Foo(...)
   *     any = Any()
   *     any.Pack(foo)
   *     if any.Is(Foo.DESCRIPTOR):
   *       any.Unpack(foo)
   *  Example 4: Pack and unpack a message in Go
   *      foo := &pb.Foo{...}
   *      any, err := anypb.New(foo)
   *      if err != nil {
   *        ...
   *      }
   *      ...
   *      foo := &pb.Foo{}
   *      if err := any.UnmarshalTo(foo); err != nil {
   * The pack methods provided by protobuf library will by default use
   * 'type.googleapis.com/full.type.name' as the type URL and the unpack
   * methods only use the fully qualified type name after the last '/'
   * in the type URL, for example "foo.bar.com/x/y.z" will yield type
   * name "y.z".
   * JSON
   * ====
   * The JSON representation of an `Any` value uses the regular
   * representation of the deserialized, embedded message, with an
   * additional field `@type` which contains the type URL. Example:
   *     package google.profile;
   *     message Person {
   *       string first_name = 1;
   *       string last_name = 2;
   *     {
   *       "@type": "type.googleapis.com/google.profile.Person",
   *       "firstName": <string>,
   *       "lastName": <string>
   * If the embedded message type is well-known and has a custom JSON
   * representation, that representation will be embedded adding a field
   * `value` which holds the custom JSON in addition to the `@type`
   * field. Example (for message [google.protobuf.Duration][]):
   *       "@type": "type.googleapis.com/google.protobuf.Duration",
   *       "value": "1.212s"
   */
  account?: ProtobufAny;
}

/**
* QueryModuleAccountsResponse is the response type for the Query/ModuleAccounts RPC method.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1QueryModuleAccountsResponse {
  accounts?: ProtobufAny[];
}

/**
 * QueryParamsResponse is the response type for the Query/Params RPC method.
 */
export interface V1Beta1QueryParamsResponse {
  /** params defines the parameters of the module. */
  params?: Authv1Beta1Params;
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
 * @title cosmos/auth/v1beta1/auth.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * @description Since: cosmos-sdk 0.47
   *
   * @tags Query
   * @name QueryAccountInfo
   * @summary AccountInfo queries account info which is common to all account types.
   * @request GET:/cosmos/auth/v1beta1/account_info/{address}
   */
  queryAccountInfo = (address: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryAccountInfoResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/account_info/${address}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description When called from another module, this query might consume a high amount of gas if the pagination field is incorrectly set. Since: cosmos-sdk 0.43
   *
   * @tags Query
   * @name QueryAccounts
   * @summary Accounts returns all the existing accounts.
   * @request GET:/cosmos/auth/v1beta1/accounts
   */
  queryAccounts = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryAccountsResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/accounts`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryAccount
   * @summary Account returns account details based on address.
   * @request GET:/cosmos/auth/v1beta1/accounts/{address}
   */
  queryAccount = (address: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryAccountResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/accounts/${address}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46.2
   *
   * @tags Query
   * @name QueryAccountAddressById
   * @summary AccountAddressByID returns account address based on account number.
   * @request GET:/cosmos/auth/v1beta1/address_by_id/{id}
   */
  queryAccountAddressByID = (id: string, query?: { account_id?: string }, params: RequestParams = {}) =>
    this.request<V1Beta1QueryAccountAddressByIDResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/address_by_id/${id}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46
   *
   * @tags Query
   * @name QueryBech32Prefix
   * @summary Bech32Prefix queries bech32Prefix
   * @request GET:/cosmos/auth/v1beta1/bech32
   */
  queryBech32Prefix = (params: RequestParams = {}) =>
    this.request<V1Beta1Bech32PrefixResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/bech32`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46
   *
   * @tags Query
   * @name QueryAddressBytesToString
   * @summary AddressBytesToString converts Account Address bytes to string
   * @request GET:/cosmos/auth/v1beta1/bech32/{address_bytes}
   */
  queryAddressBytesToString = (addressBytes: string, params: RequestParams = {}) =>
    this.request<V1Beta1AddressBytesToStringResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/bech32/${addressBytes}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46
   *
   * @tags Query
   * @name QueryAddressStringToBytes
   * @summary AddressStringToBytes converts Address string to bytes
   * @request GET:/cosmos/auth/v1beta1/bech32/{address_string}
   */
  queryAddressStringToBytes = (addressString: string, params: RequestParams = {}) =>
    this.request<V1Beta1AddressStringToBytesResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/bech32/${addressString}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46
   *
   * @tags Query
   * @name QueryModuleAccounts
   * @summary ModuleAccounts returns all the existing module accounts.
   * @request GET:/cosmos/auth/v1beta1/module_accounts
   */
  queryModuleAccounts = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryModuleAccountsResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/module_accounts`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryModuleAccountByName
   * @summary ModuleAccountByName returns the module account info by module name
   * @request GET:/cosmos/auth/v1beta1/module_accounts/{name}
   */
  queryModuleAccountByName = (name: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryModuleAccountByNameResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/module_accounts/${name}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryParams
   * @summary Params queries all parameters.
   * @request GET:/cosmos/auth/v1beta1/params
   */
  queryParams = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryParamsResponse, RpcStatus>({
      path: `/cosmos/auth/v1beta1/params`,
      method: "GET",
      format: "json",
      ...params,
    });
}
