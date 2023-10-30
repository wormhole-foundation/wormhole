import type { BaseContract, BigNumberish, BytesLike, FunctionFragment, Result, Interface, EventFragment, AddressLike, ContractRunner, ContractMethod, Listener } from 'ethers';
import type { TypedContractEvent, TypedDeferredTopicFilter, TypedEventLog, TypedLogDescription, TypedListener, TypedContractMethod } from './common';
export declare namespace BridgeStructs {
    type TransferStruct = {
        payloadID: BigNumberish;
        amount: BigNumberish;
        tokenAddress: BytesLike;
        tokenChain: BigNumberish;
        to: BytesLike;
        toChain: BigNumberish;
        fee: BigNumberish;
    };
    type TransferStructOutput = [
        payloadID: bigint,
        amount: bigint,
        tokenAddress: string,
        tokenChain: bigint,
        to: string,
        toChain: bigint,
        fee: bigint
    ] & {
        payloadID: bigint;
        amount: bigint;
        tokenAddress: string;
        tokenChain: bigint;
        to: string;
        toChain: bigint;
        fee: bigint;
    };
    type AssetMetaStruct = {
        payloadID: BigNumberish;
        tokenAddress: BytesLike;
        tokenChain: BigNumberish;
        decimals: BigNumberish;
        symbol: BytesLike;
        name: BytesLike;
    };
    type AssetMetaStructOutput = [
        payloadID: bigint,
        tokenAddress: string,
        tokenChain: bigint,
        decimals: bigint,
        symbol: string,
        name: string
    ] & {
        payloadID: bigint;
        tokenAddress: string;
        tokenChain: bigint;
        decimals: bigint;
        symbol: string;
        name: string;
    };
    type TransferWithPayloadStruct = {
        payloadID: BigNumberish;
        amount: BigNumberish;
        tokenAddress: BytesLike;
        tokenChain: BigNumberish;
        to: BytesLike;
        toChain: BigNumberish;
        fromAddress: BytesLike;
        payload: BytesLike;
    };
    type TransferWithPayloadStructOutput = [
        payloadID: bigint,
        amount: bigint,
        tokenAddress: string,
        tokenChain: bigint,
        to: string,
        toChain: bigint,
        fromAddress: string,
        payload: string
    ] & {
        payloadID: bigint;
        amount: bigint;
        tokenAddress: string;
        tokenChain: bigint;
        to: string;
        toChain: bigint;
        fromAddress: string;
        payload: string;
    };
    type RecoverChainIdStruct = {
        module: BytesLike;
        action: BigNumberish;
        evmChainId: BigNumberish;
        newChainId: BigNumberish;
    };
    type RecoverChainIdStructOutput = [
        module: string,
        action: bigint,
        evmChainId: bigint,
        newChainId: bigint
    ] & {
        module: string;
        action: bigint;
        evmChainId: bigint;
        newChainId: bigint;
    };
    type RegisterChainStruct = {
        module: BytesLike;
        action: BigNumberish;
        chainId: BigNumberish;
        emitterChainID: BigNumberish;
        emitterAddress: BytesLike;
    };
    type RegisterChainStructOutput = [
        module: string,
        action: bigint,
        chainId: bigint,
        emitterChainID: bigint,
        emitterAddress: string
    ] & {
        module: string;
        action: bigint;
        chainId: bigint;
        emitterChainID: bigint;
        emitterAddress: string;
    };
    type UpgradeContractStruct = {
        module: BytesLike;
        action: BigNumberish;
        chainId: BigNumberish;
        newContract: BytesLike;
    };
    type UpgradeContractStructOutput = [
        module: string,
        action: bigint,
        chainId: bigint,
        newContract: string
    ] & {
        module: string;
        action: bigint;
        chainId: bigint;
        newContract: string;
    };
}
export interface BridgeInterface extends Interface {
    getFunction(nameOrSignature: 'WETH' | '_parseTransferCommon' | 'attestToken' | 'bridgeContracts' | 'chainId' | 'completeTransfer' | 'completeTransferAndUnwrapETH' | 'completeTransferAndUnwrapETHWithPayload' | 'completeTransferWithPayload' | 'createWrapped' | 'encodeAssetMeta' | 'encodeTransfer' | 'encodeTransferWithPayload' | 'evmChainId' | 'finality' | 'governanceActionIsConsumed' | 'governanceChainId' | 'governanceContract' | 'isFork' | 'isInitialized' | 'isTransferCompleted' | 'isWrappedAsset' | 'outstandingBridged' | 'parseAssetMeta' | 'parsePayloadID' | 'parseRecoverChainId' | 'parseRegisterChain' | 'parseTransfer' | 'parseTransferWithPayload' | 'parseUpgrade' | 'registerChain' | 'submitRecoverChainId' | 'tokenImplementation' | 'transferTokens' | 'transferTokensWithPayload' | 'updateWrapped' | 'upgrade' | 'wormhole' | 'wrapAndTransferETH' | 'wrapAndTransferETHWithPayload' | 'wrappedAsset'): FunctionFragment;
    getEvent(nameOrSignatureOrTopic: 'AdminChanged' | 'BeaconUpgraded' | 'ContractUpgraded' | 'TransferRedeemed' | 'Upgraded'): EventFragment;
    encodeFunctionData(functionFragment: 'WETH', values?: undefined): string;
    encodeFunctionData(functionFragment: '_parseTransferCommon', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'attestToken', values: [AddressLike, BigNumberish]): string;
    encodeFunctionData(functionFragment: 'bridgeContracts', values: [BigNumberish]): string;
    encodeFunctionData(functionFragment: 'chainId', values?: undefined): string;
    encodeFunctionData(functionFragment: 'completeTransfer', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'completeTransferAndUnwrapETH', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'completeTransferAndUnwrapETHWithPayload', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'completeTransferWithPayload', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'createWrapped', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'encodeAssetMeta', values: [BridgeStructs.AssetMetaStruct]): string;
    encodeFunctionData(functionFragment: 'encodeTransfer', values: [BridgeStructs.TransferStruct]): string;
    encodeFunctionData(functionFragment: 'encodeTransferWithPayload', values: [BridgeStructs.TransferWithPayloadStruct]): string;
    encodeFunctionData(functionFragment: 'evmChainId', values?: undefined): string;
    encodeFunctionData(functionFragment: 'finality', values?: undefined): string;
    encodeFunctionData(functionFragment: 'governanceActionIsConsumed', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'governanceChainId', values?: undefined): string;
    encodeFunctionData(functionFragment: 'governanceContract', values?: undefined): string;
    encodeFunctionData(functionFragment: 'isFork', values?: undefined): string;
    encodeFunctionData(functionFragment: 'isInitialized', values: [AddressLike]): string;
    encodeFunctionData(functionFragment: 'isTransferCompleted', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'isWrappedAsset', values: [AddressLike]): string;
    encodeFunctionData(functionFragment: 'outstandingBridged', values: [AddressLike]): string;
    encodeFunctionData(functionFragment: 'parseAssetMeta', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parsePayloadID', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parseRecoverChainId', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parseRegisterChain', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parseTransfer', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parseTransferWithPayload', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'parseUpgrade', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'registerChain', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'submitRecoverChainId', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'tokenImplementation', values?: undefined): string;
    encodeFunctionData(functionFragment: 'transferTokens', values: [
        AddressLike,
        BigNumberish,
        BigNumberish,
        BytesLike,
        BigNumberish,
        BigNumberish
    ]): string;
    encodeFunctionData(functionFragment: 'transferTokensWithPayload', values: [
        AddressLike,
        BigNumberish,
        BigNumberish,
        BytesLike,
        BigNumberish,
        BytesLike
    ]): string;
    encodeFunctionData(functionFragment: 'updateWrapped', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'upgrade', values: [BytesLike]): string;
    encodeFunctionData(functionFragment: 'wormhole', values?: undefined): string;
    encodeFunctionData(functionFragment: 'wrapAndTransferETH', values: [BigNumberish, BytesLike, BigNumberish, BigNumberish]): string;
    encodeFunctionData(functionFragment: 'wrapAndTransferETHWithPayload', values: [BigNumberish, BytesLike, BigNumberish, BytesLike]): string;
    encodeFunctionData(functionFragment: 'wrappedAsset', values: [BigNumberish, BytesLike]): string;
    decodeFunctionResult(functionFragment: 'WETH', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: '_parseTransferCommon', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'attestToken', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'bridgeContracts', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'chainId', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'completeTransfer', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'completeTransferAndUnwrapETH', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'completeTransferAndUnwrapETHWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'completeTransferWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'createWrapped', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'encodeAssetMeta', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'encodeTransfer', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'encodeTransferWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'evmChainId', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'finality', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'governanceActionIsConsumed', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'governanceChainId', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'governanceContract', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'isFork', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'isInitialized', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'isTransferCompleted', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'isWrappedAsset', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'outstandingBridged', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseAssetMeta', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parsePayloadID', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseRecoverChainId', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseRegisterChain', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseTransfer', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseTransferWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'parseUpgrade', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'registerChain', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'submitRecoverChainId', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'tokenImplementation', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'transferTokens', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'transferTokensWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'updateWrapped', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'upgrade', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'wormhole', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'wrapAndTransferETH', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'wrapAndTransferETHWithPayload', data: BytesLike): Result;
    decodeFunctionResult(functionFragment: 'wrappedAsset', data: BytesLike): Result;
}
export declare namespace AdminChangedEvent {
    type InputTuple = [previousAdmin: AddressLike, newAdmin: AddressLike];
    type OutputTuple = [previousAdmin: string, newAdmin: string];
    interface OutputObject {
        previousAdmin: string;
        newAdmin: string;
    }
    type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>;
    type Filter = TypedDeferredTopicFilter<Event>;
    type Log = TypedEventLog<Event>;
    type LogDescription = TypedLogDescription<Event>;
}
export declare namespace BeaconUpgradedEvent {
    type InputTuple = [beacon: AddressLike];
    type OutputTuple = [beacon: string];
    interface OutputObject {
        beacon: string;
    }
    type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>;
    type Filter = TypedDeferredTopicFilter<Event>;
    type Log = TypedEventLog<Event>;
    type LogDescription = TypedLogDescription<Event>;
}
export declare namespace ContractUpgradedEvent {
    type InputTuple = [oldContract: AddressLike, newContract: AddressLike];
    type OutputTuple = [oldContract: string, newContract: string];
    interface OutputObject {
        oldContract: string;
        newContract: string;
    }
    type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>;
    type Filter = TypedDeferredTopicFilter<Event>;
    type Log = TypedEventLog<Event>;
    type LogDescription = TypedLogDescription<Event>;
}
export declare namespace TransferRedeemedEvent {
    type InputTuple = [
        emitterChainId: BigNumberish,
        emitterAddress: BytesLike,
        sequence: BigNumberish
    ];
    type OutputTuple = [
        emitterChainId: bigint,
        emitterAddress: string,
        sequence: bigint
    ];
    interface OutputObject {
        emitterChainId: bigint;
        emitterAddress: string;
        sequence: bigint;
    }
    type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>;
    type Filter = TypedDeferredTopicFilter<Event>;
    type Log = TypedEventLog<Event>;
    type LogDescription = TypedLogDescription<Event>;
}
export declare namespace UpgradedEvent {
    type InputTuple = [implementation: AddressLike];
    type OutputTuple = [implementation: string];
    interface OutputObject {
        implementation: string;
    }
    type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>;
    type Filter = TypedDeferredTopicFilter<Event>;
    type Log = TypedEventLog<Event>;
    type LogDescription = TypedLogDescription<Event>;
}
export interface Bridge extends BaseContract {
    connect(runner?: ContractRunner | null): Bridge;
    waitForDeployment(): Promise<this>;
    interface: BridgeInterface;
    queryFilter<TCEvent extends TypedContractEvent>(event: TCEvent, fromBlockOrBlockhash?: string | number | undefined, toBlock?: string | number | undefined): Promise<Array<TypedEventLog<TCEvent>>>;
    queryFilter<TCEvent extends TypedContractEvent>(filter: TypedDeferredTopicFilter<TCEvent>, fromBlockOrBlockhash?: string | number | undefined, toBlock?: string | number | undefined): Promise<Array<TypedEventLog<TCEvent>>>;
    on<TCEvent extends TypedContractEvent>(event: TCEvent, listener: TypedListener<TCEvent>): Promise<this>;
    on<TCEvent extends TypedContractEvent>(filter: TypedDeferredTopicFilter<TCEvent>, listener: TypedListener<TCEvent>): Promise<this>;
    once<TCEvent extends TypedContractEvent>(event: TCEvent, listener: TypedListener<TCEvent>): Promise<this>;
    once<TCEvent extends TypedContractEvent>(filter: TypedDeferredTopicFilter<TCEvent>, listener: TypedListener<TCEvent>): Promise<this>;
    listeners<TCEvent extends TypedContractEvent>(event: TCEvent): Promise<Array<TypedListener<TCEvent>>>;
    listeners(eventName?: string): Promise<Array<Listener>>;
    removeAllListeners<TCEvent extends TypedContractEvent>(event?: TCEvent): Promise<this>;
    WETH: TypedContractMethod<[], [string], 'view'>;
    _parseTransferCommon: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferStructOutput
    ], 'view'>;
    attestToken: TypedContractMethod<[
        tokenAddress: AddressLike,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    bridgeContracts: TypedContractMethod<[
        chainId_: BigNumberish
    ], [
        string
    ], 'view'>;
    chainId: TypedContractMethod<[], [bigint], 'view'>;
    completeTransfer: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        void
    ], 'nonpayable'>;
    completeTransferAndUnwrapETH: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        void
    ], 'nonpayable'>;
    completeTransferAndUnwrapETHWithPayload: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        string
    ], 'nonpayable'>;
    completeTransferWithPayload: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        string
    ], 'nonpayable'>;
    createWrapped: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        string
    ], 'nonpayable'>;
    encodeAssetMeta: TypedContractMethod<[
        meta: BridgeStructs.AssetMetaStruct
    ], [
        string
    ], 'view'>;
    encodeTransfer: TypedContractMethod<[
        transfer: BridgeStructs.TransferStruct
    ], [
        string
    ], 'view'>;
    encodeTransferWithPayload: TypedContractMethod<[
        transfer: BridgeStructs.TransferWithPayloadStruct
    ], [
        string
    ], 'view'>;
    evmChainId: TypedContractMethod<[], [bigint], 'view'>;
    finality: TypedContractMethod<[], [bigint], 'view'>;
    governanceActionIsConsumed: TypedContractMethod<[
        hash: BytesLike
    ], [
        boolean
    ], 'view'>;
    governanceChainId: TypedContractMethod<[], [bigint], 'view'>;
    governanceContract: TypedContractMethod<[], [string], 'view'>;
    isFork: TypedContractMethod<[], [boolean], 'view'>;
    isInitialized: TypedContractMethod<[impl: AddressLike], [boolean], 'view'>;
    isTransferCompleted: TypedContractMethod<[
        hash: BytesLike
    ], [
        boolean
    ], 'view'>;
    isWrappedAsset: TypedContractMethod<[token: AddressLike], [boolean], 'view'>;
    outstandingBridged: TypedContractMethod<[
        token: AddressLike
    ], [
        bigint
    ], 'view'>;
    parseAssetMeta: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.AssetMetaStructOutput
    ], 'view'>;
    parsePayloadID: TypedContractMethod<[encoded: BytesLike], [bigint], 'view'>;
    parseRecoverChainId: TypedContractMethod<[
        encodedRecoverChainId: BytesLike
    ], [
        BridgeStructs.RecoverChainIdStructOutput
    ], 'view'>;
    parseRegisterChain: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.RegisterChainStructOutput
    ], 'view'>;
    parseTransfer: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferStructOutput
    ], 'view'>;
    parseTransferWithPayload: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferWithPayloadStructOutput
    ], 'view'>;
    parseUpgrade: TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.UpgradeContractStructOutput
    ], 'view'>;
    registerChain: TypedContractMethod<[
        encodedVM: BytesLike
    ], [
        void
    ], 'nonpayable'>;
    submitRecoverChainId: TypedContractMethod<[
        encodedVM: BytesLike
    ], [
        void
    ], 'nonpayable'>;
    tokenImplementation: TypedContractMethod<[], [string], 'view'>;
    transferTokens: TypedContractMethod<[
        token: AddressLike,
        amount: BigNumberish,
        recipientChain: BigNumberish,
        recipient: BytesLike,
        arbiterFee: BigNumberish,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    transferTokensWithPayload: TypedContractMethod<[
        token: AddressLike,
        amount: BigNumberish,
        recipientChain: BigNumberish,
        recipient: BytesLike,
        nonce: BigNumberish,
        payload: BytesLike
    ], [
        bigint
    ], 'payable'>;
    updateWrapped: TypedContractMethod<[
        encodedVm: BytesLike
    ], [
        string
    ], 'nonpayable'>;
    upgrade: TypedContractMethod<[encodedVM: BytesLike], [void], 'nonpayable'>;
    wormhole: TypedContractMethod<[], [string], 'view'>;
    wrapAndTransferETH: TypedContractMethod<[
        recipientChain: BigNumberish,
        recipient: BytesLike,
        arbiterFee: BigNumberish,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    wrapAndTransferETHWithPayload: TypedContractMethod<[
        recipientChain: BigNumberish,
        recipient: BytesLike,
        nonce: BigNumberish,
        payload: BytesLike
    ], [
        bigint
    ], 'payable'>;
    wrappedAsset: TypedContractMethod<[
        tokenChainId: BigNumberish,
        tokenAddress: BytesLike
    ], [
        string
    ], 'view'>;
    getFunction<T extends ContractMethod = ContractMethod>(key: string | FunctionFragment): T;
    getFunction(nameOrSignature: 'WETH'): TypedContractMethod<[], [string], 'view'>;
    getFunction(nameOrSignature: '_parseTransferCommon'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'attestToken'): TypedContractMethod<[
        tokenAddress: AddressLike,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    getFunction(nameOrSignature: 'bridgeContracts'): TypedContractMethod<[chainId_: BigNumberish], [string], 'view'>;
    getFunction(nameOrSignature: 'chainId'): TypedContractMethod<[], [bigint], 'view'>;
    getFunction(nameOrSignature: 'completeTransfer'): TypedContractMethod<[encodedVm: BytesLike], [void], 'nonpayable'>;
    getFunction(nameOrSignature: 'completeTransferAndUnwrapETH'): TypedContractMethod<[encodedVm: BytesLike], [void], 'nonpayable'>;
    getFunction(nameOrSignature: 'completeTransferAndUnwrapETHWithPayload'): TypedContractMethod<[encodedVm: BytesLike], [string], 'nonpayable'>;
    getFunction(nameOrSignature: 'completeTransferWithPayload'): TypedContractMethod<[encodedVm: BytesLike], [string], 'nonpayable'>;
    getFunction(nameOrSignature: 'createWrapped'): TypedContractMethod<[encodedVm: BytesLike], [string], 'nonpayable'>;
    getFunction(nameOrSignature: 'encodeAssetMeta'): TypedContractMethod<[
        meta: BridgeStructs.AssetMetaStruct
    ], [
        string
    ], 'view'>;
    getFunction(nameOrSignature: 'encodeTransfer'): TypedContractMethod<[
        transfer: BridgeStructs.TransferStruct
    ], [
        string
    ], 'view'>;
    getFunction(nameOrSignature: 'encodeTransferWithPayload'): TypedContractMethod<[
        transfer: BridgeStructs.TransferWithPayloadStruct
    ], [
        string
    ], 'view'>;
    getFunction(nameOrSignature: 'evmChainId'): TypedContractMethod<[], [bigint], 'view'>;
    getFunction(nameOrSignature: 'finality'): TypedContractMethod<[], [bigint], 'view'>;
    getFunction(nameOrSignature: 'governanceActionIsConsumed'): TypedContractMethod<[hash: BytesLike], [boolean], 'view'>;
    getFunction(nameOrSignature: 'governanceChainId'): TypedContractMethod<[], [bigint], 'view'>;
    getFunction(nameOrSignature: 'governanceContract'): TypedContractMethod<[], [string], 'view'>;
    getFunction(nameOrSignature: 'isFork'): TypedContractMethod<[], [boolean], 'view'>;
    getFunction(nameOrSignature: 'isInitialized'): TypedContractMethod<[impl: AddressLike], [boolean], 'view'>;
    getFunction(nameOrSignature: 'isTransferCompleted'): TypedContractMethod<[hash: BytesLike], [boolean], 'view'>;
    getFunction(nameOrSignature: 'isWrappedAsset'): TypedContractMethod<[token: AddressLike], [boolean], 'view'>;
    getFunction(nameOrSignature: 'outstandingBridged'): TypedContractMethod<[token: AddressLike], [bigint], 'view'>;
    getFunction(nameOrSignature: 'parseAssetMeta'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.AssetMetaStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'parsePayloadID'): TypedContractMethod<[encoded: BytesLike], [bigint], 'view'>;
    getFunction(nameOrSignature: 'parseRecoverChainId'): TypedContractMethod<[
        encodedRecoverChainId: BytesLike
    ], [
        BridgeStructs.RecoverChainIdStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'parseRegisterChain'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.RegisterChainStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'parseTransfer'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'parseTransferWithPayload'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.TransferWithPayloadStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'parseUpgrade'): TypedContractMethod<[
        encoded: BytesLike
    ], [
        BridgeStructs.UpgradeContractStructOutput
    ], 'view'>;
    getFunction(nameOrSignature: 'registerChain'): TypedContractMethod<[encodedVM: BytesLike], [void], 'nonpayable'>;
    getFunction(nameOrSignature: 'submitRecoverChainId'): TypedContractMethod<[encodedVM: BytesLike], [void], 'nonpayable'>;
    getFunction(nameOrSignature: 'tokenImplementation'): TypedContractMethod<[], [string], 'view'>;
    getFunction(nameOrSignature: 'transferTokens'): TypedContractMethod<[
        token: AddressLike,
        amount: BigNumberish,
        recipientChain: BigNumberish,
        recipient: BytesLike,
        arbiterFee: BigNumberish,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    getFunction(nameOrSignature: 'transferTokensWithPayload'): TypedContractMethod<[
        token: AddressLike,
        amount: BigNumberish,
        recipientChain: BigNumberish,
        recipient: BytesLike,
        nonce: BigNumberish,
        payload: BytesLike
    ], [
        bigint
    ], 'payable'>;
    getFunction(nameOrSignature: 'updateWrapped'): TypedContractMethod<[encodedVm: BytesLike], [string], 'nonpayable'>;
    getFunction(nameOrSignature: 'upgrade'): TypedContractMethod<[encodedVM: BytesLike], [void], 'nonpayable'>;
    getFunction(nameOrSignature: 'wormhole'): TypedContractMethod<[], [string], 'view'>;
    getFunction(nameOrSignature: 'wrapAndTransferETH'): TypedContractMethod<[
        recipientChain: BigNumberish,
        recipient: BytesLike,
        arbiterFee: BigNumberish,
        nonce: BigNumberish
    ], [
        bigint
    ], 'payable'>;
    getFunction(nameOrSignature: 'wrapAndTransferETHWithPayload'): TypedContractMethod<[
        recipientChain: BigNumberish,
        recipient: BytesLike,
        nonce: BigNumberish,
        payload: BytesLike
    ], [
        bigint
    ], 'payable'>;
    getFunction(nameOrSignature: 'wrappedAsset'): TypedContractMethod<[
        tokenChainId: BigNumberish,
        tokenAddress: BytesLike
    ], [
        string
    ], 'view'>;
    getEvent(key: 'AdminChanged'): TypedContractEvent<AdminChangedEvent.InputTuple, AdminChangedEvent.OutputTuple, AdminChangedEvent.OutputObject>;
    getEvent(key: 'BeaconUpgraded'): TypedContractEvent<BeaconUpgradedEvent.InputTuple, BeaconUpgradedEvent.OutputTuple, BeaconUpgradedEvent.OutputObject>;
    getEvent(key: 'ContractUpgraded'): TypedContractEvent<ContractUpgradedEvent.InputTuple, ContractUpgradedEvent.OutputTuple, ContractUpgradedEvent.OutputObject>;
    getEvent(key: 'TransferRedeemed'): TypedContractEvent<TransferRedeemedEvent.InputTuple, TransferRedeemedEvent.OutputTuple, TransferRedeemedEvent.OutputObject>;
    getEvent(key: 'Upgraded'): TypedContractEvent<UpgradedEvent.InputTuple, UpgradedEvent.OutputTuple, UpgradedEvent.OutputObject>;
    filters: {
        'AdminChanged(address,address)': TypedContractEvent<AdminChangedEvent.InputTuple, AdminChangedEvent.OutputTuple, AdminChangedEvent.OutputObject>;
        AdminChanged: TypedContractEvent<AdminChangedEvent.InputTuple, AdminChangedEvent.OutputTuple, AdminChangedEvent.OutputObject>;
        'BeaconUpgraded(address)': TypedContractEvent<BeaconUpgradedEvent.InputTuple, BeaconUpgradedEvent.OutputTuple, BeaconUpgradedEvent.OutputObject>;
        BeaconUpgraded: TypedContractEvent<BeaconUpgradedEvent.InputTuple, BeaconUpgradedEvent.OutputTuple, BeaconUpgradedEvent.OutputObject>;
        'ContractUpgraded(address,address)': TypedContractEvent<ContractUpgradedEvent.InputTuple, ContractUpgradedEvent.OutputTuple, ContractUpgradedEvent.OutputObject>;
        ContractUpgraded: TypedContractEvent<ContractUpgradedEvent.InputTuple, ContractUpgradedEvent.OutputTuple, ContractUpgradedEvent.OutputObject>;
        'TransferRedeemed(uint16,bytes32,uint64)': TypedContractEvent<TransferRedeemedEvent.InputTuple, TransferRedeemedEvent.OutputTuple, TransferRedeemedEvent.OutputObject>;
        TransferRedeemed: TypedContractEvent<TransferRedeemedEvent.InputTuple, TransferRedeemedEvent.OutputTuple, TransferRedeemedEvent.OutputObject>;
        'Upgraded(address)': TypedContractEvent<UpgradedEvent.InputTuple, UpgradedEvent.OutputTuple, UpgradedEvent.OutputObject>;
        Upgraded: TypedContractEvent<UpgradedEvent.InputTuple, UpgradedEvent.OutputTuple, UpgradedEvent.OutputObject>;
    };
}
