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
 * Params defines the set of params for the distribution module.
 */
export interface Distributionv1Beta1Params {
  community_tax?: string;

  /**
   * Deprecated: The base_proposer_reward field is deprecated and is no longer used
   * in the x/distribution module's reward mechanism.
   */
  base_proposer_reward?: string;

  /**
   * Deprecated: The bonus_proposer_reward field is deprecated and is no longer used
   * in the x/distribution module's reward mechanism.
   */
  bonus_proposer_reward?: string;
  withdraw_addr_enabled?: boolean;
}

/**
* ValidatorOutstandingRewards represents outstanding (un-withdrawn) rewards
for a validator inexpensive to track, allows simple sanity checks.
*/
export interface Distributionv1Beta1ValidatorOutstandingRewards {
  rewards?: V1Beta1DecCoin[];
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
* DecCoin defines a token with a denomination and a decimal amount.

NOTE: The amount field is an Dec which implements the custom method
signatures required by gogoproto.
*/
export interface V1Beta1DecCoin {
  denom?: string;
  amount?: string;
}

/**
* DelegationDelegatorReward represents the properties
of a delegator's delegation reward.
*/
export interface V1Beta1DelegationDelegatorReward {
  validator_address?: string;
  reward?: V1Beta1DecCoin[];
}

/**
* MsgCommunityPoolSpendResponse defines the response to executing a
MsgCommunityPoolSpend message.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgCommunityPoolSpendResponse = object;

/**
 * MsgFundCommunityPoolResponse defines the Msg/FundCommunityPool response type.
 */
export type V1Beta1MsgFundCommunityPoolResponse = object;

/**
* MsgSetWithdrawAddressResponse defines the Msg/SetWithdrawAddress response
type.
*/
export type V1Beta1MsgSetWithdrawAddressResponse = object;

/**
* MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.

Since: cosmos-sdk 0.47
*/
export type V1Beta1MsgUpdateParamsResponse = object;

/**
* MsgWithdrawDelegatorRewardResponse defines the Msg/WithdrawDelegatorReward
response type.
*/
export interface V1Beta1MsgWithdrawDelegatorRewardResponse {
  /** Since: cosmos-sdk 0.46 */
  amount?: V1Beta1Coin[];
}

/**
* MsgWithdrawValidatorCommissionResponse defines the
Msg/WithdrawValidatorCommission response type.
*/
export interface V1Beta1MsgWithdrawValidatorCommissionResponse {
  /** Since: cosmos-sdk 0.46 */
  amount?: V1Beta1Coin[];
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
* QueryCommunityPoolResponse is the response type for the Query/CommunityPool
RPC method.
*/
export interface V1Beta1QueryCommunityPoolResponse {
  /** pool defines community pool's coins. */
  pool?: V1Beta1DecCoin[];
}

/**
* QueryDelegationRewardsResponse is the response type for the
Query/DelegationRewards RPC method.
*/
export interface V1Beta1QueryDelegationRewardsResponse {
  /** rewards defines the rewards accrued by a delegation. */
  rewards?: V1Beta1DecCoin[];
}

/**
* QueryDelegationTotalRewardsResponse is the response type for the
Query/DelegationTotalRewards RPC method.
*/
export interface V1Beta1QueryDelegationTotalRewardsResponse {
  /** rewards defines all the rewards accrued by a delegator. */
  rewards?: V1Beta1DelegationDelegatorReward[];

  /** total defines the sum of all the rewards. */
  total?: V1Beta1DecCoin[];
}

/**
* QueryDelegatorValidatorsResponse is the response type for the
Query/DelegatorValidators RPC method.
*/
export interface V1Beta1QueryDelegatorValidatorsResponse {
  /** validators defines the validators a delegator is delegating for. */
  validators?: string[];
}

/**
* QueryDelegatorWithdrawAddressResponse is the response type for the
Query/DelegatorWithdrawAddress RPC method.
*/
export interface V1Beta1QueryDelegatorWithdrawAddressResponse {
  /** withdraw_address defines the delegator address to query for. */
  withdraw_address?: string;
}

/**
 * QueryParamsResponse is the response type for the Query/Params RPC method.
 */
export interface V1Beta1QueryParamsResponse {
  /** params defines the parameters of the module. */
  params?: Distributionv1Beta1Params;
}

export interface V1Beta1QueryValidatorCommissionResponse {
  /** commission defines the commission the validator received. */
  commission?: V1Beta1ValidatorAccumulatedCommission;
}

/**
 * QueryValidatorDistributionInfoResponse is the response type for the Query/ValidatorDistributionInfo RPC method.
 */
export interface V1Beta1QueryValidatorDistributionInfoResponse {
  /** operator_address defines the validator operator address. */
  operator_address?: string;

  /** self_bond_rewards defines the self delegations rewards. */
  self_bond_rewards?: V1Beta1DecCoin[];

  /** commission defines the commission the validator received. */
  commission?: V1Beta1DecCoin[];
}

/**
* QueryValidatorOutstandingRewardsResponse is the response type for the
Query/ValidatorOutstandingRewards RPC method.
*/
export interface V1Beta1QueryValidatorOutstandingRewardsResponse {
  /**
   * ValidatorOutstandingRewards represents outstanding (un-withdrawn) rewards
   * for a validator inexpensive to track, allows simple sanity checks.
   */
  rewards?: Distributionv1Beta1ValidatorOutstandingRewards;
}

/**
* QueryValidatorSlashesResponse is the response type for the
Query/ValidatorSlashes RPC method.
*/
export interface V1Beta1QueryValidatorSlashesResponse {
  /** slashes defines the slashes the validator received. */
  slashes?: V1Beta1ValidatorSlashEvent[];

  /** pagination defines the pagination in the response. */
  pagination?: V1Beta1PageResponse;
}

/**
* ValidatorAccumulatedCommission represents accumulated commission
for a validator kept as a running counter, can be withdrawn at any time.
*/
export interface V1Beta1ValidatorAccumulatedCommission {
  commission?: V1Beta1DecCoin[];
}

/**
* ValidatorSlashEvent represents a validator slash event.
Height is implicit within the store key.
This is needed to calculate appropriate amount of staking tokens
for delegations which are withdrawn after a slash has occurred.
*/
export interface V1Beta1ValidatorSlashEvent {
  /** @format uint64 */
  validator_period?: string;
  fraction?: string;
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
 * @title cosmos/distribution/v1beta1/distribution.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryCommunityPool
   * @summary CommunityPool queries the community pool coins.
   * @request GET:/cosmos/distribution/v1beta1/community_pool
   */
  queryCommunityPool = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryCommunityPoolResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/community_pool`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryDelegationTotalRewards
 * @summary DelegationTotalRewards queries the total rewards accrued by a each
validator.
 * @request GET:/cosmos/distribution/v1beta1/delegators/{delegator_address}/rewards
 */
  queryDelegationTotalRewards = (delegatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDelegationTotalRewardsResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryDelegationRewards
   * @summary DelegationRewards queries the total rewards accrued by a delegation.
   * @request GET:/cosmos/distribution/v1beta1/delegators/{delegator_address}/rewards/{validator_address}
   */
  queryDelegationRewards = (delegatorAddress: string, validatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDelegationRewardsResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards/${validatorAddress}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryDelegatorValidators
   * @summary DelegatorValidators queries the validators of a delegator.
   * @request GET:/cosmos/distribution/v1beta1/delegators/{delegator_address}/validators
   */
  queryDelegatorValidators = (delegatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDelegatorValidatorsResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/validators`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryDelegatorWithdrawAddress
   * @summary DelegatorWithdrawAddress queries withdraw address of a delegator.
   * @request GET:/cosmos/distribution/v1beta1/delegators/{delegator_address}/withdraw_address
   */
  queryDelegatorWithdrawAddress = (delegatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryDelegatorWithdrawAddressResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/withdraw_address`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryParams
   * @summary Params queries params of the distribution module.
   * @request GET:/cosmos/distribution/v1beta1/params
   */
  queryParams = (params: RequestParams = {}) =>
    this.request<V1Beta1QueryParamsResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/params`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryValidatorDistributionInfo
   * @summary ValidatorDistributionInfo queries validator commission and self-delegation rewards for validator
   * @request GET:/cosmos/distribution/v1beta1/validators/{validator_address}
   */
  queryValidatorDistributionInfo = (validatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryValidatorDistributionInfoResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/validators/${validatorAddress}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryValidatorCommission
   * @summary ValidatorCommission queries accumulated commission for a validator.
   * @request GET:/cosmos/distribution/v1beta1/validators/{validator_address}/commission
   */
  queryValidatorCommission = (validatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryValidatorCommissionResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/validators/${validatorAddress}/commission`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryValidatorOutstandingRewards
   * @summary ValidatorOutstandingRewards queries rewards of a validator address.
   * @request GET:/cosmos/distribution/v1beta1/validators/{validator_address}/outstanding_rewards
   */
  queryValidatorOutstandingRewards = (validatorAddress: string, params: RequestParams = {}) =>
    this.request<V1Beta1QueryValidatorOutstandingRewardsResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/validators/${validatorAddress}/outstanding_rewards`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryValidatorSlashes
   * @summary ValidatorSlashes queries slash events of a validator.
   * @request GET:/cosmos/distribution/v1beta1/validators/{validator_address}/slashes
   */
  queryValidatorSlashes = (
    validatorAddress: string,
    query?: {
      starting_height?: string;
      ending_height?: string;
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1QueryValidatorSlashesResponse, RpcStatus>({
      path: `/cosmos/distribution/v1beta1/validators/${validatorAddress}/slashes`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });
}
