// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/MajorityVoting.sol";

contract MajorityVotingMock {
    function voting(bool[] memory arr) external pure returns (bool) {
        return MajorityVoting.voting(arr);
    }
}
