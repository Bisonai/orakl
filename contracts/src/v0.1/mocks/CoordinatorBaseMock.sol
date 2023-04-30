// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../CoordinatorBase.sol";

contract CoordinatorBaseMock is CoordinatorBase {
    function computeFee(uint64 reqCount) external view returns (uint256) {
        return calculateServiceFee(reqCount);
    }

    function pendingRequestExists(
        address /*consumer*/,
        uint64 /*accId*/,
        uint64 /*nonce*/
    ) external pure returns (bool) {
        return false;
    }
}
