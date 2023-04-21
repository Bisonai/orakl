// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

library MajorityVoting {
    function calculateBool(bool[] memory list) {
        uint256 trueCount = 0;
        uint256 falseCount = 0;

        for (uint256 i; i < list.length; ++i) {
            if (list[i] == true) {
                trueCount++;
            } else {
                falseCount++;
            }
        }

        if (trueCount >= falseCount) {
            return true;
        } else {
            return false;
        }
    }
}
