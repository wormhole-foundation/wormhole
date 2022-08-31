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
        uint16 chain = chainId();

        // Wormhole chain ids explicitly enumerated
        if        (chain == 2)  { tokenContract = 0x44bD47a8Bc18398227d6f40E1693Cf897bb9855E; // ethereum
        } else if (chain == 4)  { tokenContract = 0x1877a83023A87849D89a076466531b6a5DEa7eb2; // bsc
        } else if (chain == 5)  { tokenContract = 0xB9A1c8873a7a36c2Eb6D8c1A19702106CdAE6edd; // polygon
        } else if (chain == 6)  { tokenContract = 0x276a65900C97A3726319742e74F75bC4f56A0BfD; // avalanche
        } else if (chain == 7)  { tokenContract = 0x95BeDdFba786Aa1A5b3294aa6166cB125B961e34; // oasis
        } else if (chain == 9)  { tokenContract = 0x1Cd0b07Dc82482f057b3cf19775e8453309c5356; // aurora
        } else if (chain == 10) { tokenContract = 0x40D0A808241cafd9D70700963d205FeA9c0B1C9D; // fantom
        } else if (chain == 11) { tokenContract = 0x9002933919Aa83c38D01bDfBd788A9dfF42f3880; // karura
        // Acala EVM was down at the time of this migration
        // } else if (chain == 12) { tokenContract = 0x0000000000000000000000000000000000000000; // acala
        } else if (chain == 13) { tokenContract = 0xA7601785478622E720d41454CB390852cd2B9788; // klaytn
        } else if (chain == 14) { tokenContract = 0xADE06bc75Dc1FC3fB7442e0CFb8Ca544B23aF789; // celo
        } else {
            revert("Unknown chain id.");
        }

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
