// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./interfaces/IRegistry.sol";

abstract contract L1EndpointBase {
    IRegistry public REGISTRY;
    struct RequestDetail {
        uint256 l2RequestId;
        address sender;
        uint256 callbackGasLimit;
    }
    mapping(address => bool) sOracles;
    mapping(uint256 => RequestDetail) sRequest;

    error OnlyOracle();
    error InsufficientBalance();
    error ConsumerValid();

    constructor(address registry) {
        REGISTRY = IRegistry(registry);
    }
}
