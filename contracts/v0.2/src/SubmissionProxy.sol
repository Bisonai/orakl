// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IAggregator} from "./interfaces/IAggregatorSubmit.sol";

// TODO: submission verification
// TODO: submission by aggregator name
contract SubmissionProxy is Ownable {
    uint256 public constant MIN_SUBMISSION = 0;
    uint256 public constant MAX_SUBMISSION = 1_000;
    uint256 public constant MIN_EXPIRATION = 1 days;
    uint256 public constant MAX_EXPIRATION = 365 days;

    uint256 public maxSubmission = 50;
    uint256 public expirationPeriod = 5 weeks;
    mapping(address oracle => uint256 expiration) oracles;

    event OracleAdded(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();
    error InvalidExpirationPeriod();
    error InvalidMaxSubmission();

    modifier onlyOracle() {
        uint256 expiration = oracles[msg.sender];
        if (expiration == 0 || expiration <= block.timestamp) revert OnlyOracle();
        _;
    }

    constructor() Ownable(msg.sender) {}

    function setMaxSubmission(uint256 _maxSubmission) external onlyOwner {
	if (_maxSubmission == MIN_SUBMISSION || _maxSubmission > MAX_SUBMISSION) {
	    revert InvalidMaxSubmission();
	}
        maxSubmission = _maxSubmission;
        emit MaxSubmissionSet(_maxSubmission);
    }

    function setExpirationPeriod(uint256 _expirationPeriod) external onlyOwner {
	if (_expirationPeriod < MIN_EXPIRATION || _expirationPeriod > MAX_EXPIRATION) {
	    revert InvalidExpirationPeriod();
	}
        expirationPeriod = _expirationPeriod;
        emit ExpirationPeriodSet(_expirationPeriod);
    }

    function addOracle(address _oracle) external onlyOwner {
	if (oracles[_oracle] != 0) {
	    revert InvalidOracle();
	}

        oracles[_oracle] = block.timestamp + expirationPeriod;
        emit OracleAdded(_oracle);
    }

    function submit(address[] memory _aggregators, int256[] memory _submissions) external onlyOracle {
        if (_aggregators.length != _submissions.length || _aggregators.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _aggregators.length; ++i) {
            IAggregator(_aggregators[i]).submit(_submissions[i]);
        }
    }
}
