// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/Median.sol";

contract MedianMock {
    function median(int256[] memory arr) external pure returns (int256) {
        return Median.calculate(arr);
    }
}
