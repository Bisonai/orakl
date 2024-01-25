// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IAggregatorSubmit.sol";

contract BatchSubmission is Ownable {
    uint256 maxSubmission = 50;
    address[] public oracleAddresses;

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);

    constructor() {}

    function getOracles() public view returns (address[] memory) {
        return oracleAddresses;
    }

    function addOracle(address _oracle) public onlyOwner {
        oracleAddresses.push(_oracle);
        emit OracleAdded(_oracle);
    }

    function removeOracle(address _oracle) public onlyOwner {
        bool isOracleRemoved = false;
        for (uint256 i = 0; i < oracleAddresses.length; ++i) {
            if (oracleAddresses[i] == _oracle) {
                address last = oracleAddresses[oracleAddresses.length - 1];
                oracleAddresses[i] = last;
                oracleAddresses.pop();
                isOracleRemoved = true;
                break;
            }
        }
        if (isOracleRemoved) emit OracleRemoved(_oracle);
        else revert InvalidOracle();
    }

    function setMaxSubmission(uint256 _maxSubmission) public onlyOwner {
        maxSubmission = _maxSubmission;
    }

    modifier onlyOracle() {
        bool isOracle = false;
        for (uint256 i = 0; i < oracleAddresses.length; ++i) {
            if (oracleAddresses[i] == msg.sender) {
                isOracle = true;
                break;
            }
        }
        if (!isOracle) revert OnlyOracle();
        _;
    }

    function batchSubmit(
        address[] memory _aggregators,
        uint256[] memory _roundIds,
        int256[] memory _submissions
    ) public onlyOracle {
        if (
            !(_aggregators.length == _roundIds.length && _aggregators.length == _submissions.length)
        ) revert InvalidSubmissionLength();
        for (uint i = 0; i < _aggregators.length; i++) {
            IAggregator(_aggregators[i]).submit(_roundIds[i], _submissions[i]);
        }
    }
}
