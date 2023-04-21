// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

library MajorityVoting {
    error EvenLengthList();

    function voting(bool[] memory list) internal pure returns (bool) {
        if (list.length % 2 == 0) {
            revert EvenLengthList();
        }
        uint256 trueCount;
        uint256 falseCount;

        for (uint256 i; i < list.length; ++i) {
            if (list[i]) {
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
