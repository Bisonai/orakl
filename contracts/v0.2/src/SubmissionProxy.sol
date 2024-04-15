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
    mapping(address oracle => uint256 expirationTime) public whitelist;
    mapping(address feed => uint8 threshold) thresholds;

    event OracleAdded(address oracle, uint256 expirationTime);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);
    event ThresholdSet(address feed, uint8 threshold);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();
    error InvalidExpirationPeriod();
    error InvalidMaxSubmission();
    error InvalidThreshold();

    modifier onlyOracle() {
	if (!isWhitelisted(msg.sender))  {
            revert OnlyOracle();
        }
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
     * @notice Set the proof threshold for a feed.
     * @param _feed The address of the feed
     * @param _threshold The number of required signatures
     */
    function setProofThreshold(address _feed, uint8 _threshold) external onlyOwner {
        if (_threshold == 0) {
            revert InvalidThreshold();
        }

        thresholds[_feed] = _threshold;
        emit ThresholdSet(_feed, _threshold);
    }

    /**
     * @notice Add an oracle to the whitelist. The oracle will be able
     * to produce valid submission proofs until the expiration
     * time. This function is called only once for a single node
     * operator. Afterward, the oracle itself can update its address
     * using `updateOracle`. Address update must be done before the
     * expiration time.
     * @dev If the oracle is already in the whitelist, the function
     * will revert with `InvalidOracle` error.
     * @param _oracle The address of the oracle
     */
    function addOracle(address _oracle) external onlyOwner {
        if (whitelist[_oracle] != 0) {
            revert InvalidOracle();
        }

	uint256 expirationTime_ = block.timestamp + expirationPeriod;
	whitelist[_oracle] = expirationTime_;
	emit OracleAdded(_oracle, expirationTime_);
    }

    /**
     * @notice Update address of active oracle. The oracle will be able to
     * produce valid submission proofs until the expiration time.
     * @dev If the oracle address is already in the whitelist, the
     * function will revert with `InvalidOracle` error.
     * @param _oracle The address of the oracle
     */
    function updateOracle(address _oracle) external onlyOracle {
        if (whitelist[_oracle] != 0) {
            revert InvalidOracle();
        }

	whitelist[msg.sender] = block.timestamp; // deactivate the old oracle

	uint256 expirationTime_ = block.timestamp + expirationPeriod;
        whitelist[_oracle] = expirationTime_;
        emit OracleAdded(_oracle, expirationTime_);
    }

    /**
     * @notice Submit a batch of submissions to multiple feeds.
     * @dev If the number size of `_feeds`, `_answers`, and `_proofs`
     * is not equal, or longer than `maxSubmission`, the function will
     * revert with `InvalidSubmissionLength` error.
     * @param _feeds The addresses of the feeds
     * @param _answers The submissions
     * @param _proofs The proofs
     */
    function submit(address[] memory _feeds, int256[] memory _answers, bytes[] memory _proofs) external {
        if (_feeds.length != _answers.length || _answers.length != _proofs.length || _feeds.length > maxSubmission) {
            revert InvalidSubmissionLength();
        }

        for (uint256 feedIdx = 0; feedIdx < _feeds.length; feedIdx++) {
            bytes[] memory proofs_ = splitBytesToChunks(_proofs[feedIdx]);
            bytes32 message_ = keccak256(abi.encodePacked(_answers[feedIdx]));

            bool verified_ = false;
            uint8 verifiedSignatures_ = 0;
            uint8 requiredSignatures_ = thresholds[_feeds[feedIdx]];
            if (requiredSignatures_ == 0) {
                revert InvalidThreshold();
            }

            for (uint256 proofIdx = 0; proofIdx < proofs_.length; proofIdx++) {
                bytes memory singleProof_ = proofs_[proofIdx];
                address signer_ = recoverSigner(message_, singleProof_);
                if (isWhitelisted(signer_)) {
                    verifiedSignatures_++;
                    if (verifiedSignatures_ >= requiredSignatures_) {
                        verified_ = true;
                        break;
                    }
                }
            }

            if (!verified_) {
                // Insufficient number of signatures have been
                // verified -> do not submit!
                continue;
            }

            IFeed(_feeds[feedIdx]).submit(_answers[feedIdx]);
        }
    }

    /**
     * @notice Return the version and type of the feed.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion() external pure returns (string memory) {
        return "SubmissionProxy v0.2";
    }

    /**
     * @notice Split bytes into chunks of 65 bytes
     * @param data The bytes to be split
     * @return chunks The array of bytes chunks
     */
    function splitBytesToChunks(bytes memory data) private pure returns (bytes[] memory) {
        uint256 dataLength = data.length;
        uint256 numChunks = dataLength / 65;
        bytes[] memory chunks = new bytes[](numChunks);

        bytes32 halfChunk0;
        bytes32 halfChunk1;

        for (uint256 i = 0; i < numChunks; i++) {
            uint256 f = (i * 65) + 32;
            uint256 s = (i * 65) + 64;
            assembly {
                halfChunk0 := mload(add(data, f))
                halfChunk1 := mload(add(data, s))
            }
            chunks[i] = abi.encodePacked(halfChunk0, halfChunk1, data[(i * 65) + 64]);
        }

        return chunks;
    }

    /**
     * @notice Split signature into `v`, `r`, and `s` components
     * @param sig The signature to be split
     * @return v The `v` component of the signature
     * @return r The `r` component of the signature
     * @return s The `s` component of the signature
     */
    function splitSignature(bytes memory sig) private pure returns (uint8 v, bytes32 r, bytes32 s) {
        require(sig.length == 65);

        assembly {
            // first 32 bytes, after the length prefix
            r := mload(add(sig, 32))
            // second 32 bytes
            s := mload(add(sig, 64))
            // final byte (first byte of the next 32 bytes)
            v := byte(0, mload(add(sig, 96)))
        }
        return (v, r, s);
    }

    /**
     * @notice Recover the signer of the hash of the message
     * @param message The hash of the message
     * @param sig The signature of the message
     * @return The address of the signer
     */
    function recoverSigner(bytes32 message, bytes memory sig) private pure returns (address) {
        (uint8 v, bytes32 r, bytes32 s) = splitSignature(sig);
        return ecrecover(message, v, r, s);
    }

    /**
     * @notice Check if the signer is whitelisted
     * @dev The signer is whitelisted if the expiration period is not
     * 0 and the expiration period is greater than the current block
     * timestamp
     * @param _signer The address of the signer
     * @return `true` if the signer is whitelisted, `false` otherwise
     */
    function isWhitelisted(address _signer) private view returns (bool) {
        uint256 expiration_ = whitelist[_signer];
        if (expiration_ == 0 || expiration_ <= block.timestamp) {
            return false;
        } else {
            return true;
        }
    }
}
