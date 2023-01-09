// contracts/NFTBridgeImplementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./NFTBridge.sol";


contract NFTBridgeImplementation is NFTBridge {
    // Beacon getter for the token contracts
    function implementation() public view returns (address) {
        return tokenImplementation();
    }

    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        if (evmChainId() == 0) {
            uint256 evmChainId;
            uint16 chain = chainId();

            // Wormhole chain ids explicitly enumerated
            if        (chain == 2)  { evmChainId = 1;          // ethereum
            } else if (chain == 4)  { evmChainId = 56;         // bsc
            } else if (chain == 5)  { evmChainId = 137;        // polygon
            } else if (chain == 6)  { evmChainId = 43114;      // avalanche
            } else if (chain == 7)  { evmChainId = 42262;      // oasis
            } else if (chain == 9)  { evmChainId = 1313161554; // aurora
            } else if (chain == 10) { evmChainId = 250;        // fantom
            } else if (chain == 11) { evmChainId = 686;        // karura
            } else if (chain == 12) { evmChainId = 787;        // acala
            } else if (chain == 13) { evmChainId = 8217;       // klaytn
            } else if (chain == 14) { evmChainId = 42220;      // celo
            } else if (chain == 16) { evmChainId = 1284;       // moonbeam
            } else if (chain == 17) { evmChainId = 245022934;  // neon
            } else if (chain == 23) { evmChainId = 42161;      // arbitrum
            } else if (chain == 24) { evmChainId = 10;         // optimism
            } else if (chain == 25) { evmChainId = 100;        // gnosis
            } else {
                revert("Unknown chain id.");
            }

            setEvmChainId(evmChainId);
        }
    }

    modifier initializer() {
        address impl = ERC1967Upgrade._getImplementation();

        require(
            !isInitialized(impl),
            "already initialized"
        );

        setInitialized(impl);

        _;
    }
}
