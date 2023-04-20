// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./CoreRelayerState.sol";
import "@openzeppelin/contracts/utils/Context.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import {IWormhole} from "../../interfaces/IWormhole.sol";

contract CoreRelayerSetters is CoreRelayerState, Context {
    error InvalidEvmChainId();

    function setInitialized(address implementation) internal {
        _state.initializedImplementations[implementation] = true;
    }

    function setConsumedGovernanceAction(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    function setGovernanceChainId(uint16 chainId) internal {
        _state.provider.governanceChainId = chainId;
    }

    function setGovernanceContract(bytes32 governanceContract) internal {
        _state.provider.governanceContract = governanceContract;
    }

    function setChainId(uint16 chainId) internal {
        _state.provider.chainId = chainId;
    }

    function setWormhole(address wh) internal {
        _state.provider.wormhole = payable(wh);
    }

    function setRelayProvider(address defaultRelayProvider) internal {
        _state.defaultRelayProvider = defaultRelayProvider;
    }

    function setRegisteredCoreRelayerContract(uint16 chainId, bytes32 relayerAddress) internal {
        _state.registeredCoreRelayerContract[chainId] = relayerAddress;
    }

    function appendForwardInstruction(IWormholeRelayerInternalStructs.ForwardInstruction memory forwardInstruction) internal {
        _state.forwardInstructions.push(forwardInstruction);
    }

    function clearForwardInstructions() internal {
       delete _state.forwardInstructions;
    }

    function setContractLock(bool status) internal {
        _state.contractLock = status;
    }

    function setLockedTargetAddress(address targetAddress) internal {
        _state.targetAddress = targetAddress;
    }

    function setEvmChainId(uint256 evmChainId) internal {
        if (evmChainId != block.chainid) {
            revert InvalidEvmChainId();
        }
        _state.evmChainId = evmChainId;
    }
}
