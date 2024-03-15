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
    mapping(string => address) feeds;
    address[] public feedAddresses;

    event OracleAdded(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);
    event FeedAddressUpdated(string name, address indexed feed);
    event FeedAddressBulkUpdated(string[] names, address[] feeds);
    event FeedAddressRemoved(string name, address feed);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();
    error InvalidExpirationPeriod();
    error InvalidMaxSubmission();
    error InvalidFeed();

    modifier onlyOracle() {
        uint256 expiration_ = oracles[msg.sender];
        if (expiration_ == 0 || expiration_ <= block.timestamp) revert OnlyOracle();
        _;
    }

    constructor() Ownable(msg.sender) {}

    function getFeeds() external view returns (address[] memory) {
	return feedAddresses;
    }

    function removeFeed(string calldata _name) external onlyOwner {
	if (feeds[_name] == address(0)) {
	    revert InvalidFeed();
	}

	address feedToDelete_ = feeds[_name];
	delete feeds[_name];
	for (uint256 i = 0; i < feedAddresses.length; i++) {
	    if (feedAddresses[i] == feedToDelete_) {
		feedAddresses[i] = feedAddresses[feedAddresses.length - 1];
		feedAddresses.pop();
		break;
	    }
	}

	emit FeedAddressRemoved(_name, feedToDelete_);
    }

    function updateFeedBulk(string[] calldata _names, address[] calldata _feeds) external onlyOwner {
        require(_names.length > 0 && _names.length == _feeds.length, "invalid input");

        for (uint256 i = 0; i < _names.length; i++) {
	    updateFeed(_names[i], _feeds[i]);
        }

        emit FeedAddressBulkUpdated(_names, _feeds);
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

    // TODO compare with submitByPairName & remove
    function submit(address[] memory _feeds, int256[] memory _submissions) external onlyOracle {
        if (_feeds.length != _submissions.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _feeds.length; i++) {
            IFeed(_feeds[i]).submit(_submissions[i]);
        }
    }

    function submitByPairName(string[] memory _feeds, int256[] memory _submissions) external onlyOracle {
        if (_feeds.length != _submissions.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _feeds.length; i++) {
            IFeed(feeds[_feeds[i]]).submit(_submissions[i]);
        }
    }

    function typeAndVersion() external pure virtual override returns (string memory) {
        return "SubmissionProxy v0.2";
    }

    function updateFeed(string calldata _name, address _feed) public onlyOwner {
	if (feeds[_name] == address(0)) {
	    feedAddresses.push(_feed);
	}

        feeds[_name] = _feed;
        emit FeedAddressUpdated(_name, _feed);
    }
}
