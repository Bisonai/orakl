// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/Math.sol";

contract QuickSelectMock {
    function quickSelect(int256[] memory _a, uint256 _k) public pure returns(int256) {
        return Math.quickselect(_a, _k);
    }
}
