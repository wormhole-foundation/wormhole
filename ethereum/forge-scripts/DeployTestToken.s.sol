// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {ERC20PresetMinterPauser} from "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetMinterPauser.sol";
import {ERC721PresetMinterPauserAutoId} from "@openzeppelin/contracts/token/ERC721/presets/ERC721PresetMinterPauserAutoId.sol";
import {MockWETH9} from "../contracts/bridge/mock/MockWETH9.sol";
import {TokenImplementation} from "../contracts/bridge/token/TokenImplementation.sol";
import "forge-std/Script.sol";

contract DeployTestToken is Script {
    // DryRun - Deploy the system
    // dry run: forge script ./forge-scripts/DeployTestToken.s.sol:DeployTestToken --sig "dryRun()" --rpc-url $RPC
    function dryRun() public {
        _deploy();
    }

    // Deploy the system
    // deploy:  forge script ./forge-scripts/DeployTestToken.s.sol:DeployTestToken --sig "run()" --rpc-url $RPC --etherscan-api-key $ETHERSCAN_API_KEY --private-key $RAW_PRIVATE_KEY --broadcast --verify
    function run()
        public
        returns (
            address deployedTokenAddress,
            address deployedNFTaddress,
            address deployedWETHaddress,
            address deployedAccountantTokenAddress
        )
    {
        vm.startBroadcast();
        (
            deployedTokenAddress,
            deployedNFTaddress,
            deployedWETHaddress,
            deployedAccountantTokenAddress
        ) = _deploy();
        vm.stopBroadcast();
    }

    function _deploy()
        internal
        returns (
            address deployedTokenAddress,
            address deployedNFTaddress,
            address deployedWETHaddress,
            address deployedAccountantTokenAddress
        )
    {
        ERC20PresetMinterPauser token = new ERC20PresetMinterPauser(
            "Ethereum Test Token",
            "TKN"
        );
        console.log("Token deployed at: ", address(token));

        // mint 1000 units
        token.mint(address(this), 1_000_000_000_000_000_000_000);

        ERC721PresetMinterPauserAutoId nft = new ERC721PresetMinterPauserAutoId(
            unicode"Not an APE🐒",
            unicode"APE🐒",
            "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/"
        );

        nft.mint(address(this));
        nft.mint(address(this));

        console.log("NFT deployed at: ", address(nft));

        MockWETH9 mockWeth = new MockWETH9();

        console.log("WETH token deployed at: ", address(mockWeth));

        ERC20PresetMinterPauser accountantToken = new ERC20PresetMinterPauser(
            "Accountant Test Token",
            "GA"
        );

        console.log(
            "Accountant test token deployed at: ",
            address(accountantToken)
        );

        // mint 1000 units
        accountantToken.mint(address(this), 1_000_000_000_000_000_000_000);

        return (
            address(token),
            address(nft),
            address(mockWeth),
            address(accountantToken)
        );
    }
}
