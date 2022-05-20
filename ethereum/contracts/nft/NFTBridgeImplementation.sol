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
        uint8 finality;
        uint16 chain = chainId();

        // Wormhole chain ids explicitly enumerated
        if        (chain == 2)  { finality = 15; // ethereum
        } else if (chain == 4)  { finality = 15; // bsc
        } else if (chain == 5)  { finality = 15; // polygon
        } else if (chain == 6)  { finality = 1;  // avalanche
        } else if (chain == 7)  { finality = 1;  // oasis
        } else if (chain == 9)  { finality = 1;  // aurora
        } else if (chain == 10) { finality = 1;  // fantom
        } else if (chain == 11) { finality = 1;  // karura
        } else if (chain == 12) { finality = 1;  // acala
        } else if (chain == 13) { finality = 1;  // klaytn
        } else if (chain == 14) { finality = 1;  // celo
        } else if (chain == 16) { finality = 1;  // moonbeam
        } else if (chain == 17) { finality = 32; // neon
        } else {
            revert("Unknown chain id.");
        }

        setFinality(finality);
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
