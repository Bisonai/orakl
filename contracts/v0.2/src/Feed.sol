// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeed.sol";

/**
 * @title Orakl Network Feed
 * @author Bisonai
 * @notice A contract that stores the historical and latest answers, as well as
 * the timestamp submitted by the submitter.
 * @dev The submitted answers are expected to be submitted through a
 * `SubmissionProxy` contract.
 */
contract Feed is Ownable, IFeed {
    uint8 public immutable decimals;
    string public name;
    address public submitter;

    struct Round {
        int256 answer;
        uint256 updatedAt;
    }

    uint64 private latestRoundId;
    mapping(uint64 roundId => Round data) internal rounds;

    event FeedUpdated(int256 indexed answer);
    event SubmitterUpdated(address indexed submitter);

    error InvalidSubmitter();
    error OnlySubmitter();
    error NoDataPresent();
    error InsufficientData();
    error AnswerAboveTolerance();

    modifier onlySubmitter() {
        if (msg.sender != submitter) {
            revert OnlySubmitter();
        }
        _;
    }

    /**
     * @notice Construct a new `Feed` contract.
     * @dev The deployer of the contract will become the owner.
     * @param _decimals The number of decimals for the feed
     * @param _name The name of the feed
     * @param _submitter The address of the submitter
     */
    constructor(uint8 _decimals, string memory _name, address _submitter) Ownable(msg.sender) {
        decimals = _decimals;
        name = _name;
        submitter = _submitter;
    }

    /**
     * @inheritdoc IFeed
     */
    function typeAndVersion() external pure returns (string memory) {
        return "Feed v0.2";
    }

    /**
     * @notice Update the submitter address.
     * @param _submitter The address of the new submitter
     */
    function updateSubmitter(address _submitter) external onlyOwner {
        if (_submitter == address(0)) {
            revert InvalidSubmitter();
        }

        submitter = _submitter;
        emit SubmitterUpdated(_submitter);
    }

    /**
     * @notice Submit the answer for the current round. The round ID
     * is derived from the current round ID, and the answer together
     * with timestamp is stored in the contract. Only submitter can
     * submit the answer.
     * @param _answer The answer for the current round
     */
    function submit(int256 _answer) external onlySubmitter {
        uint64 roundId_ = latestRoundId + 1;

        rounds[roundId_].answer = _answer;
        rounds[roundId_].updatedAt = block.timestamp;

        emit FeedUpdated(_answer);
        latestRoundId = roundId_;
    }

    /**
     * @inheritdoc IFeed
     */
    function latestRoundData() external view virtual override returns (uint64, int256, uint256) {
        return getRoundData(latestRoundId);
    }

    /**
     * @inheritdoc IFeed
     */
    function latestRoundUpdatedAt() external view returns (uint256) {
        return rounds[latestRoundId].updatedAt;
    }

    /**
     * @inheritdoc IFeed
     */
    function twap(uint256 _interval, uint256 _latestUpdatedAtTolerance, int256 _minCount)
        external
        view
        returns (int256)
    {
        (uint64 latestRoundId_, int256 latestAnswer_, uint256 latestUpdatedAt_) = getRoundData(latestRoundId);

        if ((_latestUpdatedAtTolerance > 0) && ((block.timestamp - latestUpdatedAt_) > _latestUpdatedAtTolerance)) {
            revert AnswerAboveTolerance();
        }

        int256 count_ = 1;
        int256 sum_ = latestAnswer_;

        while (true) {
            if (((block.timestamp - latestUpdatedAt_) >= _interval) && (count_ >= _minCount)) {
                break;
            }

            if (latestRoundId_ == 1) {
                revert InsufficientData();
            }

            (uint64 roundId_, int256 answer_, uint256 updatedAt_) = getRoundData(latestRoundId_ - 1);
            sum_ += answer_;
            count_ += 1;
            latestRoundId_ = roundId_;
            latestUpdatedAt_ = updatedAt_;
        }

        return sum_ / count_;
    }

    /**
     * @inheritdoc IFeed
     */
    function getRoundData(uint64 _roundId) public view virtual override returns (uint64, int256, uint256) {
        Round memory r = rounds[_roundId];

        if (r.updatedAt == 0) {
            revert NoDataPresent();
        }

        return (_roundId, r.answer, r.updatedAt);
    }
}
