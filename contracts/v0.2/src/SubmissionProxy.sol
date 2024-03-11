// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IAggregator} from "./interfaces/IAggregatorSubmit.sol";

// TODO: submission verification
// TODO: submission by aggregator name
contract SubmissionProxy is Ownable {
    uint256 public maxSubmission = 50;
    uint256 public expirationPeriod = 5 weeks;

    address[] public oracles;
    mapping(address => uint256) expirations;

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();

    modifier onlyOracle() {
        uint256 expiration = expirations[msg.sender];
        if (expiration == 0 || expiration < block.timestamp) revert OnlyOracle();
        _;
    }

    constructor() Ownable(msg.sender) {}

    function getOracles() public view returns (address[] memory) {
        return oracles;
    }

    function getExpiredOracles() public view returns (address[] memory) {
        uint256 numOracles = oracles.length;
        uint256 numExpired = 0;
        address[] memory expiredFull = new address[](numOracles);

        for (uint256 i = 0; i < numOracles; ++i) {
            if (expirations[oracles[i]] < block.timestamp) {
                expiredFull[numExpired] = oracles[i];
                numExpired++;
            }
        }

        address[] memory expired = new address[](numExpired);
        for (uint256 i = 0; i < numExpired; ++i) {
            expired[i] = expiredFull[i];
        }

        return expired;
    }

    function addOracle(address _oracle) public onlyOwner {
        uint256 expiration_ = block.timestamp + expirationPeriod;
        expirations[_oracle] = expiration_;
        oracles.push(_oracle);
        emit OracleAdded(_oracle);
    }

    function removeOracle(address _oracle) public onlyOwner {
        if (expirations[_oracle] == 0) {
            revert InvalidOracle();
        }

        uint256 numOracles = oracles.length;
        for (uint256 i = 0; i < numOracles; ++i) {
            if (oracles[i] == _oracle) {
                address last = oracles[numOracles - 1];
                oracles[i] = last;
                oracles.pop();
                delete expirations[_oracle];
                emit OracleRemoved(_oracle);
                break;
            }
        }
    }

    function setMaxSubmission(uint256 _maxSubmission) public onlyOwner {
        maxSubmission = _maxSubmission;
        emit MaxSubmissionSet(_maxSubmission);
    }

    function setExpirationPeriod(uint256 _expirationPeriod) public onlyOwner {
        expirationPeriod = _expirationPeriod;
        emit ExpirationPeriodSet(_expirationPeriod);
    }

    function submit(address[] memory _aggregators, int256[] memory _submissions) public onlyOracle {
        if (_aggregators.length != _submissions.length) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _aggregators.length; ++i) {
            IAggregator(_aggregators[i]).submit(_submissions[i]);
        }
    }
}
