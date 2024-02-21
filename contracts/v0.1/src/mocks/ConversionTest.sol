// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

contract ConversionTest {
    uint128 UINT128_MAX_IN_UINT128 = 340282366920938463463374607431768211455;
    int256 UINT128_MAX_IN_INT256 = 340282366920938463463374607431768211455;

    function uint128ToInt256Test() external view {
        require(UINT128_MAX_IN_INT256 == int256(uint256(UINT128_MAX_IN_UINT128)));
    }

    function int256ToUint128Test() external view {
        require(UINT128_MAX_IN_UINT128 == uint128(uint256(UINT128_MAX_IN_INT256)));
    }
}
