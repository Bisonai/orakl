// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IAggregator {
    function updateRequestDetails(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds)
        external;

    function requestRate() external;

    function ICNCallback(bytes32 _requestId, int256 _response) external;

    function getlatestAnswer() external view returns (int256);

    function getlatestTimestamp() external view returns (uint256);

    function getAnswer(uint256 _roundId) external view returns (int256);

    function getTimestamp(uint256 _roundId) external view returns (uint256);

    function getlatestRound() external view returns (uint256);
}
