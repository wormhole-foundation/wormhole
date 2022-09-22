// contracts/Implementation.sol
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
        uint256 evmChainId;
        uint16 chain = chainId();


        // Wormhole chain ids explicitly enumerated
        if        (chain == 2)  { evmChainId = 5;          // ethereum
            setFinality(1);
        } else if (chain == 4)  { evmChainId = 97;         // bsc
        } else if (chain == 5)  { evmChainId = 80001;      // polygon
        } else if (chain == 6)  { evmChainId = 43113;      // avalanche
        } else if (chain == 7)  { evmChainId = 42261;      // oasis
        } else if (chain == 9)  { evmChainId = 1313161555; // aurora
        } else if (chain == 10) { evmChainId = 4002;       // fantom
        } else if (chain == 11) { evmChainId = 596;        // karura
        } else if (chain == 12) { evmChainId = 597;        // acala
        } else if (chain == 13) { evmChainId = 1001;       // klaytn
        } else if (chain == 14) { evmChainId = 44787;      // celo
        } else if (chain == 16) { evmChainId = 1287;       // moonbeam
        } else if (chain == 17) { evmChainId = 245022926;  // neon
        } else if (chain == 23) { evmChainId = 421611;     // arbitrum
        } else if (chain == 24) { evmChainId = 420;       // optimism
        } else if (chain == 25) { evmChainId = 77;        // gnosis
        } else if (chain == 10001) { evmChainId = 3;        // ropsten
        } else {
            revert("Unknown chain id.");
        }
        setEvmChainId(evmChainId);
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
