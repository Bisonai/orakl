// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.6/interfaces/AggregatorValidatorInterface.sol

interface IAggregatorValidator {
    function validate(uint256 previousRoundId, int256 previousAnswer, uint256 currentRoundId, int256 currentAnswer)
        external
        returns (bool);
}
