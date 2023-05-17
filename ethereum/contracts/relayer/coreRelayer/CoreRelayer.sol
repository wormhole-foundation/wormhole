// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormholeRelayer} from "../../interfaces/relayer/IWormholeRelayer.sol";

import {getDefaultRelayProviderState} from "./CoreRelayerStorage.sol";
import {CoreRelayerGovernance} from "./CoreRelayerGovernance.sol";
import {CoreRelayerSend} from "./CoreRelayerSend.sol";
import {CoreRelayerDelivery} from "./CoreRelayerDelivery.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";

//CoreRelayerGovernance inherits from ERC1967Upgrade, i.e. this is a proxy contract!
contract CoreRelayer is
  CoreRelayerGovernance,
  CoreRelayerSend,
  CoreRelayerDelivery,
  IWormholeRelayer
{
  //the only normal storage variable - everything else uses slot pattern
  //no point doing it for this one since it is entirely one-off and of no interest to the rest
  //  of the contract and it also can't accidentally be moved because we are at the bottom of
  //  the inheritance hierarchy here
  bool private initialized;

  constructor(address wormhole) CoreRelayerBase(wormhole) {}

  //needs to be called upon construction of the EC1967 proxy
  function initialize(address defaultRelayProvider) public {
    assert(!initialized);
    initialized = true;
    getDefaultRelayProviderState().defaultRelayProvider = defaultRelayProvider;
  }
}
