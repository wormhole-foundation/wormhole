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
 * Params defines the parameters for the tokenfactory module.
 */
export interface Osmosistokenfactoryv1Beta1Params {
  denom_creation_fee?: V1Beta1Coin[];

  /**
   * if denom_creation_fee is an empty array, then this field is used to add
   * more gas consumption to the base cost.
   * https://github.com/CosmWasm/token-factory/issues/11
   * @format uint64
   */
  denom_creation_gas_consume?: string;
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
* DenomAuthorityMetadata specifies metadata for addresses that have specific
capabilities over a token factory denom. Right now there is only one Admin
permission, but is planned to be extended to the future.
*/
export interface Tokenfactoryv1Beta1DenomAuthorityMetadata {
  /** Can be empty for no admin, or a valid osmosis address */
  admin?: string;
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

export type V1Beta1MsgBurnResponse = object;

/**
* MsgChangeAdminResponse defines the response structure for an executed
MsgChangeAdmin message.
*/
export type V1Beta1MsgChangeAdminResponse = object;

export interface V1Beta1MsgCreateDenomResponse {
  new_token_denom?: string;
}

export type V1Beta1MsgForceTransferResponse = object;

export type V1Beta1MsgMintResponse = object;

/**
* MsgSetDenomMetadataResponse defines the response structure for an executed
MsgSetDenomMetadata message.
*/
export type V1Beta1MsgSetDenomMetadataResponse = object;

/**
* MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgUpdateParamsResponse = object;

/**
* QueryDenomAuthorityMetadataResponse defines the response structure for the
DenomAuthorityMetadata gRPC query.
*/
export interface V1Beta1QueryDenomAuthorityMetadataResponse {
  /**
   * DenomAuthorityMetadata specifies metadata for addresses that have specific
   * capabilities over a token factory denom. Right now there is only one Admin
   * permission, but is planned to be extended to the future.
   */
  authority_metadata?: Tokenfactoryv1Beta1DenomAuthorityMetadata;
}

/**
* QueryDenomsFromCreatorRequest defines the response structure for the
DenomsFromCreator gRPC query.
*/
export interface V1Beta1QueryDenomsFromCreatorResponse {
  denoms?: string[];
}

/**
 * QueryParamsResponse is the response type for the Query/Params RPC method.
 */
export interface V1Beta1QueryParamsResponse {
  /** params defines the parameters of the module. */
  params?: Osmosistokenfactoryv1Beta1Params;
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
 * @title osmosis/tokenfactory/v1beta1/authorityMetadata.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
 * No description
 * 
 * @tags Query
 * @name QueryDenomAuthorityMetadata
 * @summary DenomAuthorityMetadata defines a gRPC query method for fetching
DenomAuthorityMetadata for a particular denom.
 * @request GET:/osmosis/tokenfactory/v1beta1/denoms/{denom}/authority_metadata
 */
  queryDenomAuthorityMetadata = (denom: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDenomAuthorityMetadataResponse, RpcStatus>({
      path: `/osmosis/tokenfactory/v1beta1/denoms/${denom}/authority_metadata`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryDenomsFromCreator
 * @summary DenomsFromCreator defines a gRPC query method for fetching all
denominations created by a specific admin/creator.
 * @request GET:/osmosis/tokenfactory/v1beta1/denoms_from_creator/{creator}
 */
  queryDenomsFromCreator = (creator: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDenomsFromCreatorResponse, RpcStatus>({
      path: `/osmosis/tokenfactory/v1beta1/denoms_from_creator/${creator}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryParams
 * @summary Params defines a gRPC query method that returns the tokenfactory module's
parameters.
 * @request GET:/osmosis/tokenfactory/v1beta1/params
 */
  queryParams = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryParamsResponse, RpcStatus>({
      path: `/osmosis/tokenfactory/v1beta1/params`,
      method: "GET",
      format: "json",
      ...params,
    });
}
