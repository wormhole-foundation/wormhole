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
* ModuleVersion specifies a module and its consensus version.

Since: cosmos-sdk 0.43
*/
export interface V1Beta1ModuleVersion {
  /** name of the app module */
  name?: string;

  /**
   * consensus version of the app module
   * @format uint64
   */
  version?: string;
}

/**
* MsgCancelUpgradeResponse is the Msg/CancelUpgrade response type.

Since: cosmos-sdk 0.46
*/
export type V1Beta1MsgCancelUpgradeResponse = object;

/**
* MsgSoftwareUpgradeResponse is the Msg/SoftwareUpgrade response type.

Since: cosmos-sdk 0.46
*/
export type V1Beta1MsgSoftwareUpgradeResponse = object;

/**
 * Plan specifies information about a planned upgrade and when it should occur.
 */
export interface V1Beta1Plan {
  /**
   * Sets the name for the upgrade. This name will be used by the upgraded
   * version of the software to apply any special "on-upgrade" commands during
   * the first BeginBlock method after the upgrade is applied. It is also used
   * to detect whether a software version can handle a given upgrade. If no
   * upgrade handler with this name has been set in the software, it will be
   * assumed that the software is out-of-date when the upgrade Time or Height is
   * reached and the software will exit.
   */
  name?: string;

  /**
   * Deprecated: Time based upgrades have been deprecated. Time based upgrade logic
   * has been removed from the SDK.
   * If this field is not empty, an error will be thrown.
   * @format date-time
   */
  time?: string;

  /**
   * The height at which the upgrade must be performed.
   * @format int64
   */
  height?: string;

  /**
   * Any application specific upgrade info to be included on-chain
   * such as a git commit that validators could automatically upgrade to
   */
  info?: string;

  /**
   * Deprecated: UpgradedClientState field has been deprecated. IBC upgrade logic has been
   * moved to the IBC module in the sub module 02-client.
   * If this field is not empty, an error will be thrown.
   */
  upgraded_client_state?: ProtobufAny;
}

/**
* QueryAppliedPlanResponse is the response type for the Query/AppliedPlan RPC
method.
*/
export interface V1Beta1QueryAppliedPlanResponse {
  /**
   * height is the block height at which the plan was applied.
   * @format int64
   */
  height?: string;
}

/**
 * Since: cosmos-sdk 0.46
 */
export interface V1Beta1QueryAuthorityResponse {
  address?: string;
}

/**
* QueryCurrentPlanResponse is the response type for the Query/CurrentPlan RPC
method.
*/
export interface V1Beta1QueryCurrentPlanResponse {
  /** plan is the current upgrade plan. */
  plan?: V1Beta1Plan;
}

/**
* QueryModuleVersionsResponse is the response type for the Query/ModuleVersions
RPC method.

Since: cosmos-sdk 0.43
*/
export interface V1Beta1QueryModuleVersionsResponse {
  /** module_versions is a list of module names with their consensus versions. */
  module_versions?: V1Beta1ModuleVersion[];
}

/**
* QueryUpgradedConsensusStateResponse is the response type for the Query/UpgradedConsensusState
RPC method.
*/
export interface V1Beta1QueryUpgradedConsensusStateResponse {
  /**
   * Since: cosmos-sdk 0.43
   * @format byte
   */
  upgraded_consensus_state?: string;
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
 * @title cosmos/upgrade/v1beta1/query.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryAppliedPlan
   * @summary AppliedPlan queries a previously applied upgrade plan by its name.
   * @request GET:/cosmos/upgrade/v1beta1/applied_plan/{name}
   */
  queryAppliedPlan = (name: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryAppliedPlanResponse, RpcStatus>({
      path: `/cosmos/upgrade/v1beta1/applied_plan/${name}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.46
   *
   * @tags Query
   * @name QueryAuthority
   * @summary Returns the account with authority to conduct upgrades
   * @request GET:/cosmos/upgrade/v1beta1/authority
   */
  queryAuthority = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryAuthorityResponse, RpcStatus>({
      path: `/cosmos/upgrade/v1beta1/authority`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryCurrentPlan
   * @summary CurrentPlan queries the current upgrade plan.
   * @request GET:/cosmos/upgrade/v1beta1/current_plan
   */
  queryCurrentPlan = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryCurrentPlanResponse, RpcStatus>({
      path: `/cosmos/upgrade/v1beta1/current_plan`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.43
   *
   * @tags Query
   * @name QueryModuleVersions
   * @summary ModuleVersions queries the list of module versions from state.
   * @request GET:/cosmos/upgrade/v1beta1/module_versions
   */
  queryModuleVersions = (query?: { module_name?: string }, params: RequestParams = {}) =>
    this.request<V1Beta1QueryModuleVersionsResponse, RpcStatus>({
      path: `/cosmos/upgrade/v1beta1/module_versions`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryUpgradedConsensusState
 * @summary UpgradedConsensusState queries the consensus state that will serve
as a trusted kernel for the next version of this chain. It will only be
stored at the last height of this chain.
UpgradedConsensusState RPC not supported with legacy querier
This rpc is deprecated now that IBC has its own replacement
(https://github.com/cosmos/ibc-go/blob/2c880a22e9f9cc75f62b527ca94aa75ce1106001/proto/ibc/core/client/v1/query.proto#L54)
 * @request GET:/cosmos/upgrade/v1beta1/upgraded_consensus_state/{last_height}
 */
  queryUpgradedConsensusState = (lastHeight: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryUpgradedConsensusStateResponse, RpcStatus>({
      path: `/cosmos/upgrade/v1beta1/upgraded_consensus_state/${lastHeight}`,
      method: "GET",
      format: "json",
      ...params,
    });
}
