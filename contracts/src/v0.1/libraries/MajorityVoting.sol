// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

library MajorityVoting {
    error EvenLengthList();

    function voting(bool[] memory list) pure internal returns(bool) {
        if (list.length % 2 == 0) {
            revert EvenLengthList();
        }
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
