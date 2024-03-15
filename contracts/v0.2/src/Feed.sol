// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeed} from "./interfaces/IFeed.sol";
import {ITypeAndVersion} from "./interfaces/ITypeAndVersion.sol";

contract Feed is Ownable, IFeed, ITypeAndVersion {
    uint64 private constant MAX_ROUND = 2 ** 64 - 1;
    int256 private constant NOT_FOUND = -1;

    uint8 public override decimals;
    string public override description;

    uint64 private latestRoundId;
    mapping(uint64 => Round) internal rounds;
    address[] private oracles;

    struct Round {
        int256 answer;
        uint64 updatedAt;
    }

    error OracleAlreadyEnabled();
    error OracleNotEnabled();
    error NoDataPresent();

    event OraclePermissionsUpdated(address indexed oracle, bool indexed whitelisted);
    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);

    constructor(uint8 _decimals, string memory _description) Ownable(msg.sender) {
        decimals = _decimals;
        description = _description;
    }

    // TODO verification
    function submit(int256 _answer) external {
        uint64 roundId_ = latestRoundId + 1;

	rounds[roundId_].answer = _answer;
        rounds[roundId_].updatedAt = uint64(block.timestamp);

	emit FeedUpdated(_answer, roundId_, block.timestamp);
	latestRoundId = roundId_;
    }

    function changeOracles(
        address[] calldata _removed,
        address[] calldata _added
    ) external onlyOwner {
        for (uint256 i = 0; i < _removed.length; i++) {
            removeOracle(_removed[i]);
        }

        for (uint256 i = 0; i < _added.length; i++) {
            addOracle(_added[i]);
        }
    }

    function getOracles() external view returns (address[] memory) {
        return oracles;
    }

    function latestRoundData()
        external
        view
        virtual
        override
        returns (uint64 roundId, int256 answer, uint256 updatedAt)
    {
        return getRoundData(latestRoundId);
    }

    function latestRoundUpdatedAt() external view returns (uint256) {
        Round storage round = rounds[latestRoundId];
        return round.updatedAt;
    }

    function getRoundData(uint64 _roundId)
        public
        view
        virtual
        override
        returns (uint64 roundId, int256 answer, uint256 updatedAt)
    {
        Round memory r = rounds[_roundId];

        if (r.updatedAt == 0 || !validRoundId(_roundId)) {
            revert NoDataPresent();
        }

        return (_roundId, r.answer, r.updatedAt);
    }

    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Feed v0.2";
    }

    function addOracle(address _oracle) private {
        if (oracleEnabled(_oracle) != NOT_FOUND) {
            revert OracleAlreadyEnabled();
        }

        oracles.push(_oracle);
        emit OraclePermissionsUpdated(_oracle, true);
    }

    function removeOracle(address _oracle) private {
	int256 oracleId = oracleEnabled(_oracle);
        if (oracleId == NOT_FOUND) {
            revert OracleNotEnabled();
        }

        address tail = oracles[oracles.length - 1];
        oracles[uint256(oracleId)] = tail;
	oracles.pop();

        emit OraclePermissionsUpdated(_oracle, false);
    }

    function oracleEnabled(address _oracle) private view returns (int256) {
	for (uint256 i = 0; i < oracles.length; i++) {
	    if (oracles[i] == _oracle) {
		return int256(i);
	    }
	}

	return NOT_FOUND;
    }

    function validRoundId(uint64 _roundId) private pure returns (bool) {
        return _roundId <= MAX_ROUND;
    }
}
