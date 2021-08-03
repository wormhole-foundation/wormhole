// contracts/Setters.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./PythState.sol";

contract PythSetters is PythState {
    function setInitialized(address implementatiom) internal {
        _state.initializedImplementations[implementatiom] = true;
    }

    function setGovernanceActionConsumed(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    function setChainId(uint16 chainId) internal {
        _state.provider.chainId = chainId;
    }

    function setGovernanceChainId(uint16 chainId) internal {
        _state.provider.governanceChainId = chainId;
    }

    function setGovernanceContract(bytes32 governanceContract) internal {
        _state.provider.governanceContract = governanceContract;
    }

    function setPyth2WormholeChainId(uint16 chainId) internal {
        _state.provider.pyth2WormholeChainId = chainId;
    }

    function setPyth2WormholeContract(bytes32 contractAddr) internal {
        _state.provider.pyth2WormholeContract = contractAddr;
    }

    function setWormhole(address wh) internal {
        _state.wormhole = payable(wh);
    }

    function setLatestAttestation(bytes32 product, uint8 priceType, PythStructs.PriceAttestation memory attestation) internal {
        _state.latestAttestations[product][priceType] = attestation;
    }
}