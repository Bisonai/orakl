// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IAggregatorSubmit.sol";

contract BatchSubmission is Ownable {
    uint256 maxSubmission = 50;
    mapping(address => bool) public oracleAddresses;

    error OnlyOracle();
    error InvalidOracle();
    error InvalidLength();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event Submited(uint256 aggreagtorAmount);

    constructor() {}

    function addOracle(address _oracle) public onlyOwner {
        oracleAddresses[_oracle] = true;
        emit OracleAdded(_oracle);
    }

    function removeOracle(address _oracle) public onlyOwner {
        if (!oracleAddresses[_oracle]) revert InvalidOracle();
        delete oracleAddresses[_oracle];
        emit OracleRemoved(_oracle);
    }

    function setMaxSubmission(uint256 _maxSubmission) public onlyOwner {
        maxSubmission = _maxSubmission;
    }

    modifier onlyOracle() {
        if (oracleAddresses[msg.sender] == false) revert OnlyOracle();
        _;
    }

    function batchSubmit(
        address[] memory _aggregators,
        uint256[] memory _roundIds,
        int256[] memory _submissions
    ) public onlyOracle {
        if (
            !(_aggregators.length == _roundIds.length && _aggregators.length == _submissions.length)
        ) revert InvalidLength();
        for (uint i = 0; i < _aggregators.length; i++) {
            IAggregator(_aggregators[i]).submit(_roundIds[i], _submissions[i]);
        }
        emit Submited(_aggregators.length);
    }
}
