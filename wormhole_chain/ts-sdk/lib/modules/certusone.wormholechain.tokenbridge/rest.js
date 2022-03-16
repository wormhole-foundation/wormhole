"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Api = exports.HttpClient = exports.ContentType = void 0;
var ContentType;
(function (ContentType) {
    ContentType["Json"] = "application/json";
    ContentType["FormData"] = "multipart/form-data";
    ContentType["UrlEncoded"] = "application/x-www-form-urlencoded";
})(ContentType = exports.ContentType || (exports.ContentType = {}));
class HttpClient {
    baseUrl = "";
    securityData = null;
    securityWorker = null;
    abortControllers = new Map();
    baseApiParams = {
        credentials: "same-origin",
        headers: {},
        redirect: "follow",
        referrerPolicy: "no-referrer",
    };
    constructor(apiConfig = {}) {
        Object.assign(this, apiConfig);
    }
    setSecurityData = (data) => {
        this.securityData = data;
    };
    addQueryParam(query, key) {
        const value = query[key];
        return (encodeURIComponent(key) +
            "=" +
            encodeURIComponent(Array.isArray(value) ? value.join(",") : typeof value === "number" ? value : `${value}`));
    }
    toQueryString(rawQuery) {
        const query = rawQuery || {};
        const keys = Object.keys(query).filter((key) => "undefined" !== typeof query[key]);
        return keys
            .map((key) => typeof query[key] === "object" && !Array.isArray(query[key])
            ? this.toQueryString(query[key])
            : this.addQueryParam(query, key))
            .join("&");
    }
    addQueryParams(rawQuery) {
        const queryString = this.toQueryString(rawQuery);
        return queryString ? `?${queryString}` : "";
    }
    contentFormatters = {
        [ContentType.Json]: (input) => input !== null && (typeof input === "object" || typeof input === "string") ? JSON.stringify(input) : input,
        [ContentType.FormData]: (input) => Object.keys(input || {}).reduce((data, key) => {
            data.append(key, input[key]);
            return data;
        }, new FormData()),
        [ContentType.UrlEncoded]: (input) => this.toQueryString(input),
    };
    mergeRequestParams(params1, params2) {
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
    createAbortSignal = (cancelToken) => {
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
    abortRequest = (cancelToken) => {
        const abortController = this.abortControllers.get(cancelToken);
        if (abortController) {
            abortController.abort();
            this.abortControllers.delete(cancelToken);
        }
    };
    request = ({ body, secure, path, type, query, format = "json", baseUrl, cancelToken, ...params }) => {
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
            const r = response;
            r.data = null;
            r.error = null;
            const data = await response[format]()
                .then((data) => {
                if (r.ok) {
                    r.data = data;
                }
                else {
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
            if (!response.ok)
                throw data;
            return data;
        });
    };
}
exports.HttpClient = HttpClient;
/**
 * @title tokenbridge/chain_registration.proto
 * @version version not set
 */
class Api extends HttpClient {
    /**
     * No description
     *
     * @tags Query
     * @name QueryChainRegistrationAll
     * @summary Queries a list of chainRegistration items.
     * @request GET:/certusone/wormholechain/tokenbridge/chainRegistration
     */
    queryChainRegistrationAll = (query, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/chainRegistration`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
    });
    /**
     * No description
     *
     * @tags Query
     * @name QueryChainRegistration
     * @summary Queries a chainRegistration by index.
     * @request GET:/certusone/wormholechain/tokenbridge/chainRegistration/{chainID}
     */
    queryChainRegistration = (chainID, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/chainRegistration/${chainID}`,
        method: "GET",
        format: "json",
        ...params,
    });
    /**
     * No description
     *
     * @tags Query
     * @name QueryCoinMetaRollbackProtectionAll
     * @summary Queries a list of coinMetaRollbackProtection items.
     * @request GET:/certusone/wormholechain/tokenbridge/coinMetaRollbackProtection
     */
    queryCoinMetaRollbackProtectionAll = (query, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/coinMetaRollbackProtection`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
    });
    /**
     * No description
     *
     * @tags Query
     * @name QueryCoinMetaRollbackProtection
     * @summary Queries a coinMetaRollbackProtection by index.
     * @request GET:/certusone/wormholechain/tokenbridge/coinMetaRollbackProtection/{index}
     */
    queryCoinMetaRollbackProtection = (index, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/coinMetaRollbackProtection/${index}`,
        method: "GET",
        format: "json",
        ...params,
    });
    /**
     * No description
     *
     * @tags Query
     * @name QueryConfig
     * @summary Queries a config by index.
     * @request GET:/certusone/wormholechain/tokenbridge/config
     */
    queryConfig = (params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/config`,
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
     * @request GET:/certusone/wormholechain/tokenbridge/replayProtection
     */
    queryReplayProtectionAll = (query, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/replayProtection`,
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
     * @request GET:/certusone/wormholechain/tokenbridge/replayProtection/{index}
     */
    queryReplayProtection = (index, params = {}) => this.request({
        path: `/certusone/wormholechain/tokenbridge/replayProtection/${index}`,
        method: "GET",
        format: "json",
        ...params,
    });
}
exports.Api = Api;
