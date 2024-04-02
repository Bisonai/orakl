// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeedSubmit.sol";

/**
 * @title Orakl Network Submission Proxy
 * @author Bisonai Labs
 * @notice A contract that allows oracles to batch submit to multiple
 * `Feed` contracts with a single transaction.
 * @dev The contract owner can set the maximum batch size in a single
 * transaction, and the expiration period for oracles. The maximum
 * batch size is set to 1,000 submission in a single transaction, and
 * the range of possible oracle expirations is between 1 and 365
 * days. The oracles that expired cannot be reused.
 */
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
        uint256 expiration_ = oracles[msg.sender];
        if (expiration_ == 0 || expiration_ <= block.timestamp) revert OnlyOracle();
        _;
    }

    /**
     * @notice Construct a new `SubmissionProxy` contract.
     * @dev The deployer of the contract will become the owner.
     */
    constructor() Ownable(msg.sender) {}

    /**
     * @notice Set the maximum number of submissions in a single transaction.
     * @param _maxSubmission The maximum number of submissions
     */
    function setMaxSubmission(uint256 _maxSubmission) external onlyOwner {
        if (_maxSubmission == MIN_SUBMISSION || _maxSubmission > MAX_SUBMISSION) {
            revert InvalidMaxSubmission();
        }
        maxSubmission = _maxSubmission;
        emit MaxSubmissionSet(_maxSubmission);
    }

    /**
     * @notice Set the expiration period for oracles.
     * @param _expirationPeriod The expiration period
     */
    function setExpirationPeriod(uint256 _expirationPeriod) external onlyOwner {
        if (_expirationPeriod < MIN_EXPIRATION || _expirationPeriod > MAX_EXPIRATION) {
            revert InvalidExpirationPeriod();
        }
        expirationPeriod = _expirationPeriod;
        emit ExpirationPeriodSet(_expirationPeriod);
    }

    /**
     * @notice Add an oracle to the whitelist.
     * @dev If the oracle is already in the whitelist, the function
     * will revert with `InvalidOracle` error.
     * @param _oracle The address of the oracle
     */
    function addOracle(address _oracle) external onlyOwner {
        if (oracles[_oracle] != 0) {
            revert InvalidOracle();
        }

        oracles[_oracle] = block.timestamp + expirationPeriod;
        emit OracleAdded(_oracle);
    }

    /**
     * @notice Remove an oracle from the whitelist.
     * @dev If the oracle is not in the whitelist, the function will
     * revert with `InvalidOracle` error. If the number size of
     * `_feeds` and `_submissions` is not equal, or longer than
     * `maxSubmission`, the function will revert with
     * `InvalidSubmissionLength` error.
     * @param _feeds The addresses of the feeds
     * @param _submissions The submissions
     */
    function submit(address[] memory _feeds, int256[] memory _submissions) external onlyOracle {
        if (_feeds.length != _submissions.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _feeds.length; i++) {
            IFeed(_feeds[i]).submit(_submissions[i]);
        }
    }

    /**
     * EXPERIMENTAL
     * @notice Submit a batch of submissions to multiple feeds.
     * @dev If the number size of `_feeds`, `_submissions`, and `_proofs`
     * is not equal, or longer than `maxSubmission`, the function will
     * revert with `InvalidSubmissionLength` error.
     * @param _feeds The addresses of the feeds
     * @param _submissions The submissions
     * @param _proofs The proofs
     */
    function submit(address[] memory _feeds, int256[] memory _submissions, bytes[] memory _proofs) external onlyOracle {
        if (_feeds.length != _submissions.length || _submissions.length != _proofs.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 i = 0; i < _feeds.length; i++) {
	    IFeed(_feeds[i]).submit(_submissions[i], _proofs[i]);
        }
    }

    /**
     * @notice Return the version and type of the feed.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion() external pure returns (string memory) {
        return "SubmissionProxy v0.2";
    }
}
