// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeed.sol";

/**
 * @title Orakl Network Feed
 * @author Bisonai Labs
 * @notice A contract that stores the historical and latest answers, and
 * the timestamp submitted by a whitelisted set of oracles. The
 * contract owner can add or remove oracles from the whitelist using
 * `changeOracles` function.
 * @dev A set of oracles is represented with `oracles` list and
 * `whitelist` mapping. The submitted answers are expected to be
 * submitted directly by whitelisted EOA oracles or through a
 * `SubmissionProxy` contract.
 */
contract Feed is Ownable, IFeed {
    uint8 public override decimals;
    string public override description;

    // round data
    struct Round {
        int256 answer;
        uint64 updatedAt;
    }

    uint64 private latestRoundId;
    mapping(uint64 => Round) internal rounds;

    // whitelisted oracles
    address[] private oracles;
    mapping(address => bool) private whitelist;

    event OraclePermissionsUpdated(address indexed oracle, bool indexed whitelisted);
    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);

    error OnlyOracle();
    error OracleAlreadyEnabled();
    error OracleNotEnabled();
    error NoDataPresent();

    modifier onlyOracle() {
        if (!whitelist[msg.sender]) {
            revert OnlyOracle();
        }
        _;
    }

    /**
     * @notice Construct a new `Feed` contract.
     * @dev The deployer of the contract will become the owner.
     * @param _decimals The number of decimals for the feed
     * @param _description The description of the feed
     */
    constructor(uint8 _decimals, string memory _description) Ownable(msg.sender) {
        decimals = _decimals;
        description = _description;
    }

    /**
     * @notice Submit the answer for the current round. The round ID
     * is derived from the current round ID, and the answer together
     * with timestamp is stored in the contract. Only whitelisted
     * oracles can submit the answer.
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
     * @notice Change the set of whitelisted oracles that can submit
     * answer through `submit` function.
     * @dev Only owner can call this function. The set of whitelisted
     * oracles is tracked through `oracles` list and `whitelist`
     * mapping. If an oracle is already in the whitelist, it will
     * revert with `OracleAlreadyEnabled` error, and if an oracle is
     * not in the whitelist, it will revert with `OracleNotEnabled`
     * error.
     * @param _removed The list of oracles to be removed from the
     * whitelist
     * @param _added The list of oracles to be added to the whitelist
     */
    function changeOracles(address[] calldata _removed, address[] calldata _added) external onlyOwner {
        for (uint256 i = 0; i < _removed.length; i++) {
            removeOracle(_removed[i]);
        }

        for (uint256 i = 0; i < _added.length; i++) {
            if (_added[i] == address(0)) {
                continue;
            }
            addOracle(_added[i]);
        }
    }

    /**
     * @notice Get list of whitelisted oracles
     * @return The list of whitelisted oracles
     */
    function getOracles() external view returns (address[] memory) {
        return oracles;
    }

    /**
     * @inheritdoc IFeed
     */
    function latestRoundData()
        external
        view
        virtual
        override
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
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

    /**
     * @notice Attempt to add oracle to a set of whitelisted oracles.
     * @dev If the oracle is already in the whitelist, it will revert
     * with `OracleAlreadyEnabled` error. If oracle is successfully
     * added, `OraclePermissionsUpdated` event is emitted.
     * @param _oracle The address of the oracle to be whitelisted
     */
    function addOracle(address _oracle) private {
        if (whitelist[_oracle]) {
            revert OracleAlreadyEnabled();
        }

        whitelist[_oracle] = true;
        oracles.push(_oracle);
        emit OraclePermissionsUpdated(_oracle, true);
    }

    /**
     * @notice Attempt to remove oracle from a set of whitelisted
     * oracles.
     * @dev If the oracle is not in the whitelist, it will revert with
     * `OracleNotEnabled` error. If oracle is successfully removed,
     * `OraclePermissionsUpdated` event is emitted.
     * @param _oracle The address of the oracle to be removed from the
     * whitelist
     */
    function removeOracle(address _oracle) private {
        if (!whitelist[_oracle]) {
            revert OracleNotEnabled();
        }

        whitelist[_oracle] = false;
        for (uint256 i = 0; i < oracles.length; i++) {
            if (oracles[i] == _oracle) {
                oracles[i] = oracles[oracles.length - 1];
                oracles.pop();
                break;
            }
        }

        emit OraclePermissionsUpdated(_oracle, false);
    }
}
