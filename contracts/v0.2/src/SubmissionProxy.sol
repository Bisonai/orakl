// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeedSubmit.sol";

/**
 * @title Orakl Network Submission Proxy
 * @author Bisonai
 * @notice A contract that allows oracles to batch submit to multiple
 * `Feed` contracts with a single transaction.
 * @dev The contract owner can set the maximum batch size in a single
 * transaction, and the expiration period for oracles. The maximum
 * batch size is set to 1,000 submission in a single transaction, and
 * the range of possible oracle expirations is between 1 and 365
 * days. The oracles that expired cannot be reused.
 */
contract SubmissionProxy is Ownable {
    uint256 public constant MIN_SUBMISSION = 1;
    uint256 public constant MAX_SUBMISSION = 1_000;
    uint256 public constant MIN_EXPIRATION = 1 days;
    uint256 public constant MAX_EXPIRATION = 365 days;
    uint8 public constant MIN_THRESHOLD = 1;
    uint8 public constant MAX_THRESHOLD = 100;

    uint256 public maxSubmission = 50;
    uint256 public expirationPeriod = 5 weeks;
    uint256 public dataFreshness = 10 seconds;
    uint8 public defaultThreshold = 50; // 50 %
    address[] public oracles;
    address[] public feedAddresses;

    struct OracleInfo {
        uint256 index;
        uint256 expirationTime;
    }

    mapping(address => OracleInfo) public whitelist;
    mapping(bytes32 feedHash => IFeed feed) public feeds;
    mapping(bytes32 feedHash => uint8 threshold) public thresholds;
    mapping(bytes32 feedHash => uint256 lastSubmissionTime) public lastSubmissionTimes;

    event OracleAdded(address oracle, uint256 expirationTime);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);
    event ExpirationPeriodSet(uint256 expirationPeriod);
    event DefaultThresholdSet(uint8 threshold);
    event ThresholdSet(bytes32 feedHash, uint8 threshold);
    event DataFreshnessSet(uint256 dataFreshness);
    event FeedAddressUpdated(bytes32 feedHash, address indexed feed);
    event FeedAddressBulkUpdated(bytes32[] feedHashes, address[] feeds);
    event FeedAddressRemoved(bytes32 feedHash, address feed);

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();
    error InvalidExpirationPeriod();
    error InvalidMaxSubmission();
    error InvalidThreshold();
    error IndexesNotAscending();
    error InvalidSignatureLength();
    error InvalidFeed();
    error ZeroAddressGiven();
    error AnswerOutdated();
    error AnswerSuperseded();
    error InvalidProofFormat();
    error InvalidProof();
    error FeedHashNotFound();
    error InvalidFeedHash();

    modifier onlyOracle() {
        if (!isWhitelisted(msg.sender)) {
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
     * @notice Get all feed addresses
     * @return The list of feed addresses
     */
    function getFeeds() external view returns (address[] memory) {
        return feedAddresses;
    }

    /**
     * @notice Remove feed address
     * @param _feedHash The hash of feed
     */
    function removeFeed(bytes32 _feedHash) external onlyOwner {
        IFeed feed = feeds[_feedHash];
        if (address(feed) == address(0)) {
            revert InvalidFeed();
        }

        delete feeds[_feedHash];

        uint256 feedAddressesLength = feedAddresses.length;
        for (uint256 i = 0; i < feedAddressesLength; i++) {
            if (feedAddresses[i] == address(feed)) {
                feedAddresses[i] = feedAddresses[feedAddresses.length - 1];
                feedAddresses.pop();
                break;
            }
        }

        emit FeedAddressRemoved(_feedHash, address(feed));
    }

    /**
     * @notice Update feed address
     * @param _feedHash The hash of the feed
     * @param _feed The address of the feed
     */
    function updateFeed(bytes32 _feedHash, address _feed) public onlyOwner {
        if (_feed == address(0)) {
            revert ZeroAddressGiven();
        }

        address oldFeedAddress = address(feeds[_feedHash]);

        if (oldFeedAddress == address(0)) {
            feedAddresses.push(_feed);
        } else {
            uint256 feedAddressesLength = feedAddresses.length;
            for (uint256 i = 0; i < feedAddressesLength; i++) {
                if (feedAddresses[i] == oldFeedAddress) {
                    feedAddresses[i] = _feed;
                    break;
                }
            }
        }

        feeds[_feedHash] = IFeed(_feed);
        emit FeedAddressUpdated(_feedHash, _feed);
    }

    /**
     * @notice Update feed addresses in bulk
     * @param _feedHashes The feedHashes of the feeds
     * @param _feeds The addresses of the feeds
     */
    function updateFeedBulk(bytes32[] calldata _feedHashes, address[] calldata _feeds) external onlyOwner {
        require(_feedHashes.length > 0 && _feedHashes.length == _feeds.length, "invalid input");

        for (uint256 i = 0; i < _feedHashes.length; i++) {
            updateFeed(_feedHashes[i], _feeds[i]);
        }

        emit FeedAddressBulkUpdated(_feedHashes, _feeds);
    }

    /**
     * @notice Set the maximum number of submissions in a single transaction.
     * @param _maxSubmission The maximum number of submissions
     */
    function setMaxSubmission(uint256 _maxSubmission) external onlyOwner {
        if (_maxSubmission < MIN_SUBMISSION || _maxSubmission > MAX_SUBMISSION) {
            revert InvalidMaxSubmission();
        }
        maxSubmission = _maxSubmission;
        emit MaxSubmissionSet(_maxSubmission);
    }

    /**
     * @notice Set the data freshness for oracles.
     * @param _dataFreshness The data freshness
     */
    function setDataFreshness(uint256 _dataFreshness) external onlyOwner {
        dataFreshness = _dataFreshness;
        emit DataFreshnessSet(_dataFreshness);
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
     * @notice Set the default proof threshold for feeds.
     * @param _threshold The percentage of required
     * signatures. Percentage is represented as a number between 1 and
     * 100.
     */
    function setDefaultProofThreshold(uint8 _threshold) external onlyOwner {
        if (_threshold < MIN_THRESHOLD || _threshold > MAX_THRESHOLD) {
            revert InvalidThreshold();
        }

        defaultThreshold = _threshold;
        emit DefaultThresholdSet(_threshold);
    }

    /**
     * @notice Set the proof threshold for a feed.
     * @param _feedHash The hash of the feed
     * @param _threshold The percentage of required
     * signatures. Percentage is represented as a number between 1 and
     * 100.
     */
    function setProofThreshold(bytes32 _feedHash, uint8 _threshold) external onlyOwner {
        if (_threshold < MIN_THRESHOLD || _threshold > MAX_THRESHOLD) {
            revert InvalidThreshold();
        }

        thresholds[_feedHash] = _threshold;
        emit ThresholdSet(_feedHash, _threshold);
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
     * @return The index of the oracle in the whitelist
     */
    function addOracle(address _oracle) external onlyOwner returns (uint256) {
        if (_oracle == address(0)) {
            revert ZeroAddressGiven();
        }

        if (whitelist[_oracle].expirationTime != 0) {
            revert InvalidOracle();
        }

        bool found = false;
        uint256 index_ = 0;

        // register the oracle
        uint256 oraclesLength_ = oracles.length;
        for (uint256 i = 0; i < oraclesLength_; i++) {
            if (!isWhitelisted(oracles[i])) {
                // reuse existing oracle slot if it is expired
                whitelist[oracles[i]].index = 0;
                oracles[i] = _oracle;

                found = true;
                index_ = i;
                break;
            }
        }

        if (!found) {
            // oracle has not been registered yet
            oracles.push(_oracle);
            index_ = oracles.length - 1;
        }

        // set the expiration time and index
        OracleInfo storage info = whitelist[_oracle];
        uint256 expirationTime_ = block.timestamp + expirationPeriod;
        info.expirationTime = expirationTime_;
        info.index = index_;

        emit OracleAdded(_oracle, expirationTime_);

        return index_;
    }

    /**
     * @notice Remove an oracle from the whitelist. The oracle will not be
     * able to produce valid submission proofs after the expiration
     * time.
     * @param _oracle The address of the oracle
     */
    function removeOracle(address _oracle) external onlyOwner {
        if (!isWhitelisted(_oracle)) {
            revert InvalidOracle();
        }

        uint256 oraclesLength_ = oracles.length;
        for (uint256 i = 0; i < oraclesLength_; i++) {
            if (_oracle == oracles[i]) {
                oracles[i] = oracles[oracles.length - 1];
                whitelist[oracles[i]].index = i;
                oracles.pop();
                break;
            }
        }

        OracleInfo storage info = whitelist[_oracle];
        info.index = 0;
        info.expirationTime = block.timestamp;

        emit OracleRemoved(_oracle);
    }

    /**
     * @notice Update address of active oracle. The oracle will be able to
     * produce valid submission proofs until the expiration time.
     * @dev If the oracle address is already in the whitelist, the
     * function will revert with `InvalidOracle` error.
     * @param _oracle The address of the oracle
     */
    function updateOracle(address _oracle) external onlyOracle {
        OracleInfo storage info = whitelist[_oracle];
        if (info.expirationTime != 0) {
            revert InvalidOracle();
        }

        // deactivate the old oracle
        whitelist[msg.sender].expirationTime = block.timestamp;
        whitelist[msg.sender].index = 0;

        // update the oracle address
        uint256 oraclesLength_ = oracles.length;
        for (uint256 i = 0; i < oraclesLength_; i++) {
            if (msg.sender == oracles[i]) {
                oracles[i] = _oracle;
                info.index = i;
                break;
            }
        }

        // extend the expiration time
        uint256 expirationTime_ = block.timestamp + expirationPeriod;
        info.expirationTime = expirationTime_;

        emit OracleAdded(_oracle, expirationTime_);
    }

    /**
     * @notice Submit a batch of submissions to multiple feeds.
     * @dev If the number size of `_feeds`, `_answers`, and `_proofs`
     * is not equal, or longer than `maxSubmission`, the function will
     * revert with `InvalidSubmissionLength` error. If the data are
     * outdated (older than `dataFreshness`), the function will not
     * submit the data. If the proof is invalid, the function will not
     * submit the data.
     * @param _feedHashes The hashes of the feeds
     * @param _answers The submissions
     * @param _timestamps The unixmilli timestamps of the proofs
     * @param _proofs The proofs
     */
    function submit(
        bytes32[] calldata _feedHashes,
        int256[] calldata _answers,
        uint256[] calldata _timestamps,
        bytes[] calldata _proofs
    ) external {
        if (
            _feedHashes.length != _answers.length || _answers.length != _proofs.length
                || _proofs.length != _timestamps.length || _feedHashes.length > maxSubmission
        ) {
            revert InvalidSubmissionLength();
        }

        uint256 feedsLength_ = _feedHashes.length;
        for (uint256 i = 0; i < feedsLength_; i++) {
            if (
                _timestamps[i] <= (block.timestamp - dataFreshness) * 1000
                    || lastSubmissionTimes[_feedHashes[i]] >= _timestamps[i]
            ) {
                // answer is too old -> do not submit!
                continue;
            }

            (bytes[] memory proofs_, bool success_) = splitProofs(_proofs[i]);
            if (!success_) {
                // splitting proofs failed -> do not submit!
                continue;
            }

            if (address(feeds[_feedHashes[i]]) == address(0)) {
                // feedHash not registered -> do not submit!
                continue;
            }

            if (keccak256(abi.encodePacked(feeds[_feedHashes[i]].name())) != _feedHashes[i]) {
                // feedHash not matching with registered feed -> do not submit!
                continue;
            }

            bytes32 message_ = keccak256(abi.encodePacked(_answers[i], _timestamps[i], _feedHashes[i]));
            if (validateProof(_feedHashes[i], message_, proofs_)) {
                feeds[_feedHashes[i]].submit(_answers[i]);
                lastSubmissionTimes[_feedHashes[i]] = _timestamps[i];
            }
        }
    }

    function submitStrict(
        bytes32[] calldata _feedHashes,
        int256[] calldata _answers,
        uint256[] calldata _timestamps,
        bytes[] calldata _proofs
    ) external {
        if (
            _feedHashes.length != _answers.length || _answers.length != _proofs.length
                || _proofs.length != _timestamps.length || _feedHashes.length > maxSubmission
        ) {
            revert InvalidSubmissionLength();
        }

        uint256 feedsLength_ = _feedHashes.length;
        for (uint256 i = 0; i < feedsLength_; i++) {
            submitSingle(_feedHashes[i], _answers[i], _timestamps[i], _proofs[i]);
        }
    }

    function submitSingle(bytes32 _feedHash, int256 _answer, uint256 _timestamp, bytes calldata _proof) public {
        if (_timestamp <= (block.timestamp - dataFreshness) * 1000 ) {
            revert AnswerOutdated();
        }

        if (lastSubmissionTimes[_feedHash] >= _timestamp) {
            revert AnswerSuperseded();
        }

        (bytes[] memory proofs_, bool success_) = splitProofs(_proof);
        if (!success_) {
            // splitting proofs failed -> do not submit!
            revert InvalidProofFormat();
        }

        if (address(feeds[_feedHash]) == address(0)) {
            // feedHash not registered -> do not submit!
            revert FeedHashNotFound();
        }

        if (keccak256(abi.encodePacked(feeds[_feedHash].name())) != _feedHash) {
            // feedHash not matching with registered feed -> do not submit!
            revert InvalidFeedHash();
        }

        bytes32 message_ = keccak256(abi.encodePacked(_answer, _timestamp, _feedHash));
        if (validateProof(_feedHash, message_, proofs_)) {
            feeds[_feedHash].submit(_answer);
            lastSubmissionTimes[_feedHash] = _timestamp;
        } else {
            revert InvalidProof();
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
     * @notice Split concatenated proofs into individual proofs of length 65 bytes
     * @param _data The bytes to be split
     * @return proofs_ The split bytes
     * @return success_ `true` if the split was successful, `false`
     */
    function splitProofs(bytes memory _data) internal pure returns (bytes[] memory proofs_, bool) {
        uint256 dataLength_ = _data.length;
        if (dataLength_ == 0 || dataLength_ % 65 != 0) {
            return (proofs_, false);
        }

        uint256 numProofs_ = dataLength_ / 65;
        proofs_ = new bytes[](numProofs_);

        for (uint256 i = 0; i < numProofs_; i++) {
            bytes memory chunk_ = new bytes(65);
            assembly {
                // Load the first half of the chunk
                let firstHalfPtr := add(_data, add(mul(i, 65), 32))
                mstore(add(chunk_, 32), mload(firstHalfPtr))

                // Load the second half of the chunk
                let secondHalfPtr := add(_data, add(mul(i, 65), 64))
                mstore(add(chunk_, 64), mload(secondHalfPtr))
            }
            // Copy the last byte of the chunk
            chunk_[64] = _data[i * 65 + 64];

            proofs_[i] = chunk_;
        }

        return (proofs_, true);
    }

    /**
     * @notice Split signature into `v`, `r`, and `s` components
     * @param _sig The signature to be split
     * @return v_ The `v` component of the signature
     * @return r_ The `r` component of the signature
     * @return s_ The `s` component of the signature
     */
    function splitSignature(bytes memory _sig) internal pure returns (uint8 v_, bytes32 r_, bytes32 s_) {
        if (_sig.length != 65) {
            revert InvalidSignatureLength();
        }

        assembly {
            let signature_ := add(_sig, 32)
            r_ := mload(signature_)
            s_ := mload(add(signature_, 32))
            v_ := byte(0, mload(add(signature_, 64)))
        }
    }

    /**
     * @notice Recover the signer of the hash of the message
     * @param message The hash of the message
     * @param sig The signature of the message
     * @return The address of the signer
     */
    function recoverSigner(bytes32 message, bytes memory sig) private pure returns (address) {
        (uint8 v, bytes32 r, bytes32 s) = splitSignature(sig);
        if (uint256(s) > 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0) {
            // return address(0) if s is larger than half of the order, to skip signature malleability
            return address(0);
        }
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
        uint256 expiration_ = whitelist[_signer].expirationTime;
        if (expiration_ == 0 || expiration_ <= block.timestamp) {
            return false;
        } else {
            return true;
        }
    }

    /**
     * @notice Calculate the quorum for the threshold
     * @param _threshold The threshold
     * @return The quorum
     */
    function quorum(uint8 _threshold) internal view returns (uint256) {
        uint256 nominator = oracles.length * _threshold;
        return (nominator / 100) + (nominator % 100 == 0 ? 0 : 1);
    }

    /**
     * @notice Get all oracles
     * @return The list of oracles
     */
    function getAllOracles() public view returns (address[] memory) {
        return oracles;
    }

    /**
     * @notice Validate the proof
     * @dev The order of the proofs must be in ascending order of the
     * oracle index. The function will revert with `IndexesNotAscending`
     * error if the order is not ascending.
     * @param _feedHash The hash of the feed
     * @param _message The hash of the message
     * @param _proofs The proofs
     * @return `true` if the proof is valid, `false` otherwise
     */
    function validateProof(bytes32 _feedHash, bytes32 _message, bytes[] memory _proofs) private view returns (bool) {
        if (oracles.length == 0) {
            return false;
        }

        uint256 verifiedSignatures_ = 0;
        uint256 lastIndex_ = 0;

        uint8 threshold_ = thresholds[_feedHash];
        if (threshold_ == 0) {
            threshold_ = defaultThreshold;
        }
        uint256 requiredSignatures_ = quorum(threshold_);

        uint256 proofsLength_ = _proofs.length;
        for (uint256 j = 0; j < proofsLength_; j++) {
            bytes memory proof_ = _proofs[j];
            address signer_ = recoverSigner(_message, proof_);
            if (signer_ == address(0)) {
                continue;
            }

            uint256 oracleIndex_ = whitelist[signer_].index;
            if (j != 0 && oracleIndex_ <= lastIndex_) {
                revert IndexesNotAscending();
            }
            lastIndex_ = oracleIndex_;

            if (isWhitelisted(signer_)) {
                verifiedSignatures_++;
                if (verifiedSignatures_ >= requiredSignatures_) {
                    return true;
                }
            }
        }

        return false;
    }
}
