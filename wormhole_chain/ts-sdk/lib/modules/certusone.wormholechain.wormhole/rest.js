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
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
var __rest = (this && this.__rest) || function (s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
        t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
        for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
            if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
                t[p[i]] = s[p[i]];
        }
    return t;
};
export var ContentType;
(function (ContentType) {
    ContentType["Json"] = "application/json";
    ContentType["FormData"] = "multipart/form-data";
    ContentType["UrlEncoded"] = "application/x-www-form-urlencoded";
})(ContentType || (ContentType = {}));
var HttpClient = /** @class */ (function () {
    function HttpClient(apiConfig) {
        var _a;
        var _this = this;
        if (apiConfig === void 0) { apiConfig = {}; }
        this.baseUrl = "";
        this.securityData = null;
        this.securityWorker = null;
        this.abortControllers = new Map();
        this.baseApiParams = {
            credentials: "same-origin",
            headers: {},
            redirect: "follow",
            referrerPolicy: "no-referrer",
        };
        this.setSecurityData = function (data) {
            _this.securityData = data;
        };
        this.contentFormatters = (_a = {},
            _a[ContentType.Json] = function (input) {
                return input !== null && (typeof input === "object" || typeof input === "string") ? JSON.stringify(input) : input;
            },
            _a[ContentType.FormData] = function (input) {
                return Object.keys(input || {}).reduce(function (data, key) {
                    data.append(key, input[key]);
                    return data;
                }, new FormData());
            },
            _a[ContentType.UrlEncoded] = function (input) { return _this.toQueryString(input); },
            _a);
        this.createAbortSignal = function (cancelToken) {
            if (_this.abortControllers.has(cancelToken)) {
                var abortController_1 = _this.abortControllers.get(cancelToken);
                if (abortController_1) {
                    return abortController_1.signal;
                }
                return void 0;
            }
            var abortController = new AbortController();
            _this.abortControllers.set(cancelToken, abortController);
            return abortController.signal;
        };
        this.abortRequest = function (cancelToken) {
            var abortController = _this.abortControllers.get(cancelToken);
            if (abortController) {
                abortController.abort();
                _this.abortControllers.delete(cancelToken);
            }
        };
        this.request = function (_a) {
            var body = _a.body, secure = _a.secure, path = _a.path, type = _a.type, query = _a.query, _b = _a.format, format = _b === void 0 ? "json" : _b, baseUrl = _a.baseUrl, cancelToken = _a.cancelToken, params = __rest(_a, ["body", "secure", "path", "type", "query", "format", "baseUrl", "cancelToken"]);
            var secureParams = (secure && _this.securityWorker && _this.securityWorker(_this.securityData)) || {};
            var requestParams = _this.mergeRequestParams(params, secureParams);
            var queryString = query && _this.toQueryString(query);
            var payloadFormatter = _this.contentFormatters[type || ContentType.Json];
            return fetch("".concat(baseUrl || _this.baseUrl || "").concat(path).concat(queryString ? "?".concat(queryString) : ""), __assign(__assign({}, requestParams), { headers: __assign(__assign({}, (type && type !== ContentType.FormData ? { "Content-Type": type } : {})), (requestParams.headers || {})), signal: cancelToken ? _this.createAbortSignal(cancelToken) : void 0, body: typeof body === "undefined" || body === null ? null : payloadFormatter(body) })).then(function (response) { return __awaiter(_this, void 0, void 0, function () {
                var r, data;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            r = response;
                            r.data = null;
                            r.error = null;
                            return [4 /*yield*/, response[format]()
                                    .then(function (data) {
                                    if (r.ok) {
                                        r.data = data;
                                    }
                                    else {
                                        r.error = data;
                                    }
                                    return r;
                                })
                                    .catch(function (e) {
                                    r.error = e;
                                    return r;
                                })];
                        case 1:
                            data = _a.sent();
                            if (cancelToken) {
                                this.abortControllers.delete(cancelToken);
                            }
                            if (!response.ok)
                                throw data;
                            return [2 /*return*/, data];
                    }
                });
            }); });
        };
        Object.assign(this, apiConfig);
    }
    HttpClient.prototype.addQueryParam = function (query, key) {
        var value = query[key];
        return (encodeURIComponent(key) +
            "=" +
            encodeURIComponent(Array.isArray(value) ? value.join(",") : typeof value === "number" ? value : "".concat(value)));
    };
    HttpClient.prototype.toQueryString = function (rawQuery) {
        var _this = this;
        var query = rawQuery || {};
        var keys = Object.keys(query).filter(function (key) { return "undefined" !== typeof query[key]; });
        return keys
            .map(function (key) {
            return typeof query[key] === "object" && !Array.isArray(query[key])
                ? _this.toQueryString(query[key])
                : _this.addQueryParam(query, key);
        })
            .join("&");
    };
    HttpClient.prototype.addQueryParams = function (rawQuery) {
        var queryString = this.toQueryString(rawQuery);
        return queryString ? "?".concat(queryString) : "";
    };
    HttpClient.prototype.mergeRequestParams = function (params1, params2) {
        return __assign(__assign(__assign(__assign({}, this.baseApiParams), params1), (params2 || {})), { headers: __assign(__assign(__assign({}, (this.baseApiParams.headers || {})), (params1.headers || {})), ((params2 && params2.headers) || {})) });
    };
    return HttpClient;
}());
export { HttpClient };
/**
 * @title wormhole/config.proto
 * @version version not set
 */
var Api = /** @class */ (function (_super) {
    __extends(Api, _super);
    function Api() {
        var _this = _super !== null && _super.apply(this, arguments) || this;
        /**
         * No description
         *
         * @tags Query
         * @name QueryConfig
         * @summary Queries a config by index.
         * @request GET:/certusone/wormholechain/wormhole/config
         */
        _this.queryConfig = function (params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/config", method: "GET", format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QueryGuardianSetAll
         * @summary Queries a list of guardianSet items.
         * @request GET:/certusone/wormholechain/wormhole/guardianSet
         */
        _this.queryGuardianSetAll = function (query, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/guardianSet", method: "GET", query: query, format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QueryGuardianSet
         * @summary Queries a guardianSet by index.
         * @request GET:/certusone/wormholechain/wormhole/guardianSet/{index}
         */
        _this.queryGuardianSet = function (index, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/guardianSet/".concat(index), method: "GET", format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QueryReplayProtectionAll
         * @summary Queries a list of replayProtection items.
         * @request GET:/certusone/wormholechain/wormhole/replayProtection
         */
        _this.queryReplayProtectionAll = function (query, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/replayProtection", method: "GET", query: query, format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QueryReplayProtection
         * @summary Queries a replayProtection by index.
         * @request GET:/certusone/wormholechain/wormhole/replayProtection/{index}
         */
        _this.queryReplayProtection = function (index, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/replayProtection/".concat(index), method: "GET", format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QuerySequenceCounterAll
         * @summary Queries a list of sequenceCounter items.
         * @request GET:/certusone/wormholechain/wormhole/sequenceCounter
         */
        _this.querySequenceCounterAll = function (query, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/sequenceCounter", method: "GET", query: query, format: "json" }, params));
        };
        /**
         * No description
         *
         * @tags Query
         * @name QuerySequenceCounter
         * @summary Queries a sequenceCounter by index.
         * @request GET:/certusone/wormholechain/wormhole/sequenceCounter/{index}
         */
        _this.querySequenceCounter = function (index, params) {
            if (params === void 0) { params = {}; }
            return _this.request(__assign({ path: "/certusone/wormholechain/wormhole/sequenceCounter/".concat(index), method: "GET", format: "json" }, params));
        };
        return _this;
    }
    return Api;
}(HttpClient));
export { Api };
