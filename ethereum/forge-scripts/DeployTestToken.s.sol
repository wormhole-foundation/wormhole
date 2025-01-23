// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {ERC20PresetMinterPauser} from "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetMinterPauser.sol";
import {ERC721PresetMinterPauserAutoId} from "@openzeppelin/contracts/token/ERC721/presets/ERC721PresetMinterPauserAutoId.sol";
import {MockWETH9} from "../contracts/bridge/mock/MockWETH9.sol";
import {TokenImplementation} from "../contracts/bridge/token/TokenImplementation.sol";
import "forge-std/Script.sol";

contract DeployTestToken is Script {
    function dryRun() public {
        _deploy();
    }

    function run()
        public
        returns (
            address deployedTokenAddress,
            address deployedNFTaddress,
            address deployedWETHaddress,
            address deployedAccountantTokenAddress,
            address transferVerificationTokenA
        )
    {
        vm.startBroadcast();
        (
            deployedTokenAddress,
            deployedNFTaddress,
            deployedWETHaddress,
            deployedAccountantTokenAddress,
            transferVerificationTokenA
        ) = _deploy();
        vm.stopBroadcast();
    }

    function _deploy()
        internal
        returns (
            address deployedTokenAddress,
            address deployedNFTaddress,
            address deployedWETHaddress,
            address deployedAccountantTokenAddress,
            address transferVerificationTokenA
        )
    {
        address[] memory accounts = new address[](14);
        accounts[0] = 0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1;
        accounts[1] = 0xFFcf8FDEE72ac11b5c542428B35EEF5769C409f0;
        accounts[2] = 0x22d491Bde2303f2f43325b2108D26f1eAbA1e32b;
        accounts[3] = 0xE11BA2b4D45Eaed5996Cd0823791E0C93114882d;
        accounts[4] = 0xd03ea8624C8C5987235048901fB614fDcA89b117;
        accounts[5] = 0x95cED938F7991cd0dFcb48F0a06a40FA1aF46EBC;
        accounts[6] = 0x3E5e9111Ae8eB78Fe1CC3bb8915d5D461F3Ef9A9;
        accounts[7] = 0x28a8746e75304c0780E011BEd21C72cD78cd535E;
        accounts[8] = 0xACa94ef8bD5ffEE41947b4585a84BdA5a3d3DA6E;
        accounts[9] = 0x1dF62f291b2E969fB0849d99D9Ce41e2F137006e;
        accounts[10] = 0x610Bb1573d1046FCb8A70Bbbd395754cD57C2b60;
        accounts[11] = 0x855FA758c77D68a04990E992aA4dcdeF899F654A;
        accounts[12] = 0xfA2435Eacf10Ca62ae6787ba2fB044f8733Ee843;
        accounts[13] = 0x64E078A8Aa15A41B85890265648e965De686bAE6;

        
        ERC20PresetMinterPauser token = new ERC20PresetMinterPauser(
            "Ethereum Test Token",
            "TKN"
        );
        console.log("Token deployed at: ", address(token));

        // mint 1000 units
        token.mint(accounts[0], 1_000_000_000_000_000_000_000);

        ERC721PresetMinterPauserAutoId nft = new ERC721PresetMinterPauserAutoId(
            unicode"Not an APE🐒",
            unicode"APE🐒",
            "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/"
        );

        nft.mint(accounts[0]);
        nft.mint(accounts[0]);

        console.log("NFT deployed at: ", address(nft));

        MockWETH9 mockWeth = new MockWETH9();

        console.log("WETH token deployed at: ", address(mockWeth));

        for(uint16 i=2; i<11; i++) {
            token.mint(accounts[i], 1_000_000_000_000_000_000_000);
        }

        ERC20PresetMinterPauser accountantToken = new ERC20PresetMinterPauser(
            "Accountant Test Token",
            "GA"
        );

        console.log(
            "Accountant test token deployed at: ",
            address(accountantToken)
        );

        // mint 1000 units
        accountantToken.mint(accounts[9], 1_000_000_000_000_000_000_000);

        for(uint16 i=0; i<11; i++) {
            // Give the accounts enough eth to send transactions
            vm.deal(accounts[i], 1e18);
        }

        // Deploy a test token for Transfer Verification
        ERC20PresetMinterPauser deployedA = new ERC20PresetMinterPauser(
            "TransferVerifier Test Token A",
            "TVA"
        );
        console.log("Test token A deployed at: ", address(deployedA));

        // Mint Eth and test tokens to Anvil's default account at index 13.
        // This will be used for Transfer Verification integration tests.
        // The other accounts created by Anvil are used for other tests in the devnet, so
        // using account 14 to send transfers will hopefully cause things to be
        // better encapsulated.
        deployedA.mint(accounts[13], 1_000_000_000_000_000_000_000);
        token.mint(accounts[13], 1_000_000_000_000_000_000_000);
        vm.deal(accounts[13], 1e18);

        return (
            address(token),
            address(nft),
            address(mockWeth),
            address(accountantToken),
            address(deployedA)
        );
    }
}
