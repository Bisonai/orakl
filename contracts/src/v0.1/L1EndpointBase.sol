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

    function pay(uint64 accId, address sender, uint256 fee) internal {
        if (!sOracles[msg.sender]) {
            revert OnlyOracle();
        }
        //check consumer and balance
        bool isValidConsumer = REGISTRY.isValidConsumer(accId, sender);
        if (!isValidConsumer) {
            revert ConsumerValid();
        }
        uint256 balance = REGISTRY.getBalance(accId);
        REGISTRY.decreaseBalance(accId, fee);
        if (balance < fee) {
            revert InsufficientBalance();
        }
    }
}
