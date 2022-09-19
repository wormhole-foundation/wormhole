// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./Bridge.sol";


contract BridgeImplementation is Bridge {
    // Beacon getter for the token contracts
    function implementation() public view returns (address) {
        return tokenImplementation();
    }

    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        address tokenContract;
        uint256 evmChainId;
        uint16 chain = chainId();

        // Wormhole chain ids explicitly enumerated
        if        (chain == 2)  { evmChainId = 1;          // ethereum
            setFinality(1);
            tokenContract = 0x0fD04a68d3c3A692d6Fa30384D1A87Ef93554eE6;
        } else if (chain == 4)  { evmChainId = 56;         // bsc
            tokenContract = 0x7f8C5e730121657E17E452c5a1bA3fA1eF96f22a;
        } else if (chain == 5)  { evmChainId = 137;        // polygon
            tokenContract = 0x7C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2;
        } else if (chain == 6)  { evmChainId = 43114;      // avalanche
            tokenContract = 0xe07548528D7c0C470251CF1374eF762345f298eE;
        } else if (chain == 7)  { evmChainId = 42262;      // oasis
            tokenContract = 0x75d520ed7fE263b96cCC7165aCe270097bC11721;
        } else if (chain == 9)  { evmChainId = 1313161554; // aurora
            tokenContract = 0x20F989Ad4C3B6ddcd940A66013d45f45d5c15463;
        } else if (chain == 10) { evmChainId = 250;        // fantom
            tokenContract = 0x99A3385C5AA40B184F6F6898daeBcD752C4b11F8;
        } else if (chain == 11) { evmChainId = 686;        // karura
            tokenContract = 0x7C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2;
        } else if (chain == 12) { evmChainId = 787;        // acala
        } else if (chain == 13) { evmChainId = 8217;       // klaytn
            tokenContract = 0x7Ec2f3742F5D7ecF85817D67Ae3f89fa70164e8F;
        } else if (chain == 14) { evmChainId = 42220;      // celo
            tokenContract = 0x1a81c975d0e69206a45584BB98520f25dEEC7b6C;
        } else if (chain == 16) { evmChainId = 1284;       // moonbeam
            tokenContract = 0xddA94dA500AF7DCd8DE53482a39eD55d4aA3B392;
        } else if (chain == 17) { evmChainId = 245022934;  // neon
        } else if (chain == 23) { evmChainId = 42161;      // arbitrum
        } else if (chain == 24) { evmChainId = 10;         // optimism
        } else if (chain == 25) { evmChainId = 100;        // gnosis
        } else {
            revert("Unknown chain id.");
        }

        setEvmChainId(evmChainId);
        setTokenImplementation(tokenContract);
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
