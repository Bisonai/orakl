// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeedSubmit.sol";
import {ITypeAndVersion} from "./interfaces/ITypeAndVersion.sol";

// TODO: submission verification
contract SubmissionProxy is Ownable, ITypeAndVersion {
    uint256 public constant MIN_SUBMISSION = 0;
    uint256 public constant MAX_SUBMISSION = 1_000;
    uint256 public constant MIN_EXPIRATION = 1 days;
    uint256 public constant MAX_EXPIRATION = 365 days;

    uint256 public maxSubmission = 50;
    uint256 public expirationPeriod = 5 weeks;
    mapping(address oracle => uint256 expiration) oracles;
    address[] public feedAddresses;

    event OracleAdded(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();
    error InvalidExpirationPeriod();
    error InvalidMaxSubmission();

    modifier onlyOracle() {
        uint256 expiration_ = oracles[msg.sender];
        if (expiration_ == 0 || expiration_ <= block.timestamp) revert OnlyOracle();
        _;
    }

    constructor() Ownable(msg.sender) {}

    function getFeeds() external view returns (address[] memory) {
	return feedAddresses;
    }

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

    function submit(address[] memory _feeds, int256[] memory _submissions) external onlyOracle {
        if (_feeds.length != _submissions.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _feeds.length; i++) {
            IFeed(_feeds[i]).submit(_submissions[i]);
        }
    }

    function typeAndVersion() external pure virtual override returns (string memory) {
        return "SubmissionProxy v0.2";
    }
}
