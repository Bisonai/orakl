// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeed.sol";

/**
 * @title Orakl Network Feed
 * @author Bisonai Labs
 * @notice A contract that stores the historical and latest answers, and
 * the timestamp submitted by oracle.
 * @dev The submitted answers are expected to be submitted through a
 * `SubmissionProxy` contract.
 */
contract Feed is Ownable, IFeed {
    uint8 public decimals;
    string public description;
    address public oracle;

    struct Round {
        int256 answer;
        uint64 updatedAt;
    }

    uint64 private latestRoundId;
    mapping(uint64 roundId => Round data) internal rounds;

    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);

    error OnlyOracle();
    error NoDataPresent();

    modifier onlyOracle() {
        if (msg.sender != oracle) {
            revert OnlyOracle();
        }
        _;
    }

    /**
     * @notice Construct a new `Feed` contract.
     * @dev The deployer of the contract will become the owner.
     * @param _decimals The number of decimals for the feed
     * @param _description The description of the feed
     * @param _oracle The address of the oracle
     */
    constructor(uint8 _decimals, string memory _description, address _oracle) Ownable(msg.sender) {
        decimals = _decimals;
        description = _description;
        oracle = _oracle;
    }

    /**
     * @notice Submit the answer for the current round. The round ID
     * is derived from the current round ID, and the answer together
     * with timestamp is stored in the contract. Only oracle can
     * submit the answer.
     * @param _answer The answer for the current round
     */
    function submit(int256 _answer) external onlyOracle {
        uint64 roundId_ = latestRoundId + 1;

        rounds[roundId_].answer = _answer;
        rounds[roundId_].updatedAt = uint64(block.timestamp);

        emit FeedUpdated(_answer, roundId_, block.timestamp);
        latestRoundId = roundId_;
    }

    /**
     * @inheritdoc IFeed
     */
    function latestRoundData() external view virtual override returns (uint64 id, int256 answer, uint256 updatedAt) {
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
    function typeAndVersion() external pure returns (string memory) {
        return "Feed v0.2";
    }

    /**
     * @inheritdoc IFeed
     */
    function getRoundData(uint64 _roundId)
        public
        view
        virtual
        override
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        Round memory r = rounds[_roundId];

        if (r.updatedAt == 0) {
            revert NoDataPresent();
        }

        return (_roundId, r.answer, r.updatedAt);
    }
}
