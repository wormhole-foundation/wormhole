import { txClient, queryClient, MissingWalletError, registry } from './module';
// @ts-ignore
import { SpVuexError } from '@starport/vuex';
import { Config } from "./module/types/wormhole/config";
import { EventGuardianSetUpdate } from "./module/types/wormhole/events";
import { EventPostedMessage } from "./module/types/wormhole/events";
import { GuardianSetUpdateProposal } from "./module/types/wormhole/governance";
import { GovernanceWormholeMessageProposal } from "./module/types/wormhole/governance";
import { GuardianSet } from "./module/types/wormhole/guardian_set";
import { ReplayProtection } from "./module/types/wormhole/replay_protection";
import { SequenceCounter } from "./module/types/wormhole/sequence_counter";
export { Config, EventGuardianSetUpdate, EventPostedMessage, GuardianSetUpdateProposal, GovernanceWormholeMessageProposal, GuardianSet, ReplayProtection, SequenceCounter };
async function initTxClient(vuexGetters) {
    return await txClient(vuexGetters['common/wallet/signer'], {
        addr: vuexGetters['common/env/apiTendermint']
    });
}
async function initQueryClient(vuexGetters) {
    return await queryClient({
        addr: vuexGetters['common/env/apiCosmos']
    });
}
function mergeResults(value, next_values) {
    for (let prop of Object.keys(next_values)) {
        if (Array.isArray(next_values[prop])) {
            value[prop] = [...value[prop], ...next_values[prop]];
        }
        else {
            value[prop] = next_values[prop];
        }
    }
    return value;
}
function getStructure(template) {
    let structure = { fields: [] };
    for (const [key, value] of Object.entries(template)) {
        let field = {};
        field.name = key;
        field.type = typeof value;
        structure.fields.push(field);
    }
    return structure;
}
const getDefaultState = () => {
    return {
        GuardianSet: {},
        GuardianSetAll: {},
        Config: {},
        ReplayProtection: {},
        ReplayProtectionAll: {},
        SequenceCounter: {},
        SequenceCounterAll: {},
        _Structure: {
            Config: getStructure(Config.fromPartial({})),
            EventGuardianSetUpdate: getStructure(EventGuardianSetUpdate.fromPartial({})),
            EventPostedMessage: getStructure(EventPostedMessage.fromPartial({})),
            GuardianSetUpdateProposal: getStructure(GuardianSetUpdateProposal.fromPartial({})),
            GovernanceWormholeMessageProposal: getStructure(GovernanceWormholeMessageProposal.fromPartial({})),
            GuardianSet: getStructure(GuardianSet.fromPartial({})),
            ReplayProtection: getStructure(ReplayProtection.fromPartial({})),
            SequenceCounter: getStructure(SequenceCounter.fromPartial({})),
        },
        _Registry: registry,
        _Subscriptions: new Set(),
    };
};
// initial state
const state = getDefaultState();
export default {
    namespaced: true,
    state,
    mutations: {
        RESET_STATE(state) {
            Object.assign(state, getDefaultState());
        },
        QUERY(state, { query, key, value }) {
            state[query][JSON.stringify(key)] = value;
        },
        SUBSCRIBE(state, subscription) {
            state._Subscriptions.add(JSON.stringify(subscription));
        },
        UNSUBSCRIBE(state, subscription) {
            state._Subscriptions.delete(JSON.stringify(subscription));
        }
    },
    getters: {
        getGuardianSet: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.GuardianSet[JSON.stringify(params)] ?? {};
        },
        getGuardianSetAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.GuardianSetAll[JSON.stringify(params)] ?? {};
        },
        getConfig: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.Config[JSON.stringify(params)] ?? {};
        },
        getReplayProtection: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.ReplayProtection[JSON.stringify(params)] ?? {};
        },
        getReplayProtectionAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.ReplayProtectionAll[JSON.stringify(params)] ?? {};
        },
        getSequenceCounter: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SequenceCounter[JSON.stringify(params)] ?? {};
        },
        getSequenceCounterAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SequenceCounterAll[JSON.stringify(params)] ?? {};
        },
        getTypeStructure: (state) => (type) => {
            return state._Structure[type].fields;
        },
        getRegistry: (state) => {
            return state._Registry;
        }
    },
    actions: {
        init({ dispatch, rootGetters }) {
            console.log('Vuex module: certusone.wormholechain.wormhole initialized!');
            if (rootGetters['common/env/client']) {
                rootGetters['common/env/client'].on('newblock', () => {
                    dispatch('StoreUpdate');
                });
            }
        },
        resetState({ commit }) {
            commit('RESET_STATE');
        },
        unsubscribe({ commit }, subscription) {
            commit('UNSUBSCRIBE', subscription);
        },
        async StoreUpdate({ state, dispatch }) {
            state._Subscriptions.forEach(async (subscription) => {
                try {
                    const sub = JSON.parse(subscription);
                    await dispatch(sub.action, sub.payload);
                }
                catch (e) {
                    throw new SpVuexError('Subscriptions: ' + e.message);
                }
            });
        },
        async QueryGuardianSet({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryGuardianSet(key.index)).data;
                commit('QUERY', { query: 'GuardianSet', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryGuardianSet', payload: { options: { all }, params: { ...key }, query } });
                return getters['getGuardianSet']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryGuardianSet', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryGuardianSetAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryGuardianSetAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryGuardianSetAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'GuardianSetAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryGuardianSetAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getGuardianSetAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryGuardianSetAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryConfig({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryConfig()).data;
                commit('QUERY', { query: 'Config', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryConfig', payload: { options: { all }, params: { ...key }, query } });
                return getters['getConfig']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryConfig', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryReplayProtection({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryReplayProtection(key.index)).data;
                commit('QUERY', { query: 'ReplayProtection', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryReplayProtection', payload: { options: { all }, params: { ...key }, query } });
                return getters['getReplayProtection']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryReplayProtection', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryReplayProtectionAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryReplayProtectionAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryReplayProtectionAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'ReplayProtectionAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryReplayProtectionAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getReplayProtectionAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryReplayProtectionAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySequenceCounter({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySequenceCounter(key.index)).data;
                commit('QUERY', { query: 'SequenceCounter', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySequenceCounter', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSequenceCounter']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySequenceCounter', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySequenceCounterAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySequenceCounterAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.querySequenceCounterAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'SequenceCounterAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySequenceCounterAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSequenceCounterAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySequenceCounterAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async sendMsgExecuteGovernanceVAA({ rootGetters }, { value, fee = [], memo = '' }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgExecuteGovernanceVAA(value);
                const result = await txClient.signAndBroadcast([msg], { fee: { amount: fee,
                        gas: "200000" }, memo });
                return result;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgExecuteGovernanceVAA:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgExecuteGovernanceVAA:Send', 'Could not broadcast Tx: ' + e.message);
                }
            }
        },
        async MsgExecuteGovernanceVAA({ rootGetters }, { value }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgExecuteGovernanceVAA(value);
                return msg;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgExecuteGovernanceVAA:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgExecuteGovernanceVAA:Create', 'Could not create message: ' + e.message);
                }
            }
        },
    }
};
