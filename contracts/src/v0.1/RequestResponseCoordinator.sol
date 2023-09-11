// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./interfaces/IRequestResponseCoordinatorBase.sol";
import "./RequestResponseConsumerFulfill.sol";
import "./CoordinatorBase.sol";
import "./libraries/Orakl.sol";
import "./libraries/Median.sol";
import "./libraries/MajorityVoting.sol";

/// @title Orakl Network RequestResponseCoordinator
/// @author Bisonai
/// @notice Accepts requests for off-chain data either through
/// [regular] or [temporary] account by calling `requestData`
/// function. Consumers can choose what data type (`jobId`) they want
/// to receive the requested data in, and how many oracles
/// (`numSubmission`) they want to participate on an aggregated
/// answer. Consumers can define the data source and postprocessing
/// steps that should be applied on data received from API. The request is
/// concluded by emitting `DataRequested` event which includes all
/// necessary metadata to provide the requested off-chain
/// data. Off-chain oracles that are registered within the
/// `RequestResponseCoordinator` then compete for delivering the
/// requested data back to on-chain because only a limited number of
/// oracle can submit the requested answer. Answers from off-chain oracles
/// are being collected in contract storage, and the last requested
/// off-chain oracle that submits its answer will also execute
/// consumer's fulfillment function, distributes reward to all
/// participating oracles, and cleanup the storage.
contract RequestResponseCoordinator is
    CoordinatorBase,
    IRequestResponseCoordinatorBase,
    ITypeAndVersion
{
    uint8 public constant MAX_ORACLES = 255;

    using Orakl for Orakl.Request;

    struct Submission {
        address[] oracles; // oracles that submitted response
        mapping(address => bool) submitted;
    }

    /* requestId */
    /* submission details */
    mapping(uint256 => Submission) sSubmission;

    /* oracle */
    /* registration status */
    mapping(address => bool) private sIsOracleRegistered;

    /* jobId */
    /* ability to request for the job */
    mapping(bytes32 => bool) private sJobId;

    mapping(uint256 => int256[]) private sRequestToSubmissionInt256;
    mapping(uint256 => uint128[]) private sRequestToSubmissionUint128;
    mapping(uint256 => bool[]) private sRequestToSubmissionBool;

    error TooManyOracles();
    error UnregisteredOracleFulfillment(address oracle);
    error InvalidJobId();
    error InvalidNumSubmission();
    error OracleAlreadySubmitted();
    error IncompatibleJobId();
    error InvalidAccRequest();

    event OracleRegistered(address oracle);
    event OracleDeregistered(address oracle);
    event PrepaymentSet(address prepayment);
    event DataRequested(
        uint256 indexed requestId,
        bytes32 jobId,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        address indexed sender,
        bool isDirectPayment,
        uint8 numSubmission,
        bytes data
    );
    event DataRequestFulfilledUint128(
        uint256 indexed requestId,
        uint256 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledInt256(
        uint256 indexed requestId,
        int256 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBool(
        uint256 indexed requestId,
        bool response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledString(
        uint256 indexed requestId,
        string response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBytes32(
        uint256 indexed requestId,
        bytes32 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBytes(
        uint256 indexed requestId,
        bytes response,
        uint256 payment,
        bool success
    );
    event DataSubmitted(address oracle, uint256 requestId);

    constructor(address prepayment) {
        sJobId[keccak256(abi.encodePacked("uint128"))] = true;
        sJobId[keccak256(abi.encodePacked("int256"))] = true;
        sJobId[keccak256(abi.encodePacked("bool"))] = true;
        sJobId[keccak256(abi.encodePacked("string"))] = true;
        sJobId[keccak256(abi.encodePacked("bytes32"))] = true;
        sJobId[keccak256(abi.encodePacked("bytes"))] = true;

        sPrepayment = IPrepayment(prepayment);
        emit PrepaymentSet(prepayment);
    }

    /**
     * @notice Register an oracle
     * @param oracle address of the oracle
     */
    function registerOracle(address oracle) external onlyOwner {
        if (sOracles.length >= MAX_ORACLES) {
            revert TooManyOracles();
        }

        if (sIsOracleRegistered[oracle]) {
            revert OracleAlreadyRegistered(oracle);
        }
        sOracles.push(oracle);
        sIsOracleRegistered[oracle] = true;
        emit OracleRegistered(oracle);
    }

    /**
     * @notice Deregister an oracle
     * @param oracle address of the oracle
     */
    function deregisterOracle(address oracle) external onlyOwner {
        if (!sIsOracleRegistered[oracle]) {
            revert NoSuchOracle(oracle);
        }
        delete sIsOracleRegistered[oracle];

        uint256 oraclesLength = sOracles.length;
        for (uint256 i = 0; i < oraclesLength; ++i) {
            if (sOracles[i] == oracle) {
                address last = sOracles[oraclesLength - 1];
                sOracles[i] = last;
                sOracles.pop();
                break;
            }
        }

        emit OracleDeregistered(oracle);
    }

    /**
     * @inheritdoc IRequestResponseCoordinatorBase
     */
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numSubmission
    ) external nonReentrant returns (uint256) {
        (uint256 balance, uint64 reqCount, , , IAccount.AccountType accType) = sPrepayment
            .getAccount(accId);
        bool isValidReq = sPrepayment.isValidReq(accId);
        if (!isValidReq) {
            revert InvalidAccRequest();
        }
        uint256 minBalance = estimateFeeByAcc(
            reqCount,
            numSubmission,
            callbackGasLimit,
            accId,
            accType
        );

        if (balance < minBalance) {
            revert InsufficientPayment(balance, minBalance);
        }

        bool isDirectPayment = false;
        uint256 requestId = requestData(
            req,
            accId,
            callbackGasLimit,
            numSubmission,
            isDirectPayment
        );

        return requestId;
    }

    /**
     * @inheritdoc IRequestResponseCoordinatorBase
     */
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address refundRecipient
    ) external payable nonReentrant returns (uint256) {
        uint64 reqCount = 0;
        uint256 fee = estimateFee(reqCount, numSubmission, callbackGasLimit);
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = sPrepayment.createTemporaryAccount(msg.sender);
        bool isDirectPayment = true;
        uint256 requestId = requestData(
            req,
            accId,
            callbackGasLimit,
            numSubmission,
            isDirectPayment
        );
        sPrepayment.depositTemporary{value: fee}(accId);

        // Refund extra $KLAY
        uint256 remaining = msg.value - fee;
        if (remaining > 0) {
            (bool sent, ) = refundRecipient.call{value: remaining}("");
            if (!sent) {
                revert RefundFailure();
            }
        }

        return requestId;
    }

    function fulfillDataRequestUint128(
        uint256 requestId,
        uint128 response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("uint128"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        uint128[] storage arrRes = sRequestToSubmissionUint128[requestId];
        arrRes.push(response);

        sSubmission[requestId].submitted[msg.sender] = true;
        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        if (arrRes.length < rc.numSubmission) {
            emit DataSubmitted(msg.sender, requestId);
            return;
        }

        int256[] memory responses = uint128ToInt256(arrRes);
        uint128 aggregatedResponse = uint128(uint256((Median.calculate(responses))));

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillUint128.rawFulfillDataRequest.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionUint128[requestId];
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledUint128(requestId, response, payment, success);
    }

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("int256"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        sSubmission[requestId].submitted[msg.sender] = true;
        int256[] storage arrRes = sRequestToSubmissionInt256[requestId];
        arrRes.push(response);

        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        if (arrRes.length < rc.numSubmission) {
            emit DataSubmitted(msg.sender, requestId);
            return;
        }

        int256 aggregatedResponse = Median.calculate(arrRes);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillInt256.rawFulfillDataRequest.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionInt256[requestId];
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledInt256(requestId, response, payment, success);
    }

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("bool"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        sSubmission[requestId].submitted[msg.sender] = true;
        bool[] storage arrRes = sRequestToSubmissionBool[requestId];
        arrRes.push(response);

        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        if (arrRes.length < rc.numSubmission) {
            emit DataSubmitted(msg.sender, requestId);
            return;
        }

        bool aggregatedResponse = MajorityVoting.voting(arrRes);
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBool.rawFulfillDataRequest.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionBool[requestId];
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledBool(requestId, response, payment, success);
    }

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("string"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        sSubmission[requestId].submitted[msg.sender] = true;
        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillString.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledString(requestId, response, payment, success);
    }

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("bytes32"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        sSubmission[requestId].submitted[msg.sender] = true;
        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes32.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledBytes32(requestId, response, payment, success);
    }

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc
    ) external nonReentrant {
        uint256 startGas = gasleft();
        if (rc.jobId != keccak256(abi.encodePacked("bytes"))) {
            revert IncompatibleJobId();
        }
        validateDataResponse(rc, requestId);

        sSubmission[requestId].submitted[msg.sender] = true;
        address[] storage oracles = sSubmission[requestId].oracles;
        oracles.push(msg.sender);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        address[] memory oraclesToPay = cleanupAfterFulfillment(requestId);
        uint256 payment = pay(rc, startGas, oraclesToPay);

        emit DataRequestFulfilledBytes(requestId, response, payment, success);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "RequestResponseCoordinator v0.1";
    }

    /**
     * @notice Find out whether given oracle address was registered.
     * @return true when oracle address registered, otherwise false
     */
    function isOracleRegistered(address oracle) external view returns (bool) {
        return sIsOracleRegistered[oracle];
    }

    /**
     * @inheritdoc IRequestResponseCoordinatorBase
     */
    function validateNumSubmission(bytes32 jobId, uint8 numSubmission) public view {
        if (!sJobId[jobId]) {
            revert InvalidJobId();
        }

        if (numSubmission == 0) {
            revert InvalidNumSubmission();
        } else if (jobId == keccak256(abi.encodePacked("bool")) && numSubmission % 2 == 0) {
            revert InvalidNumSubmission();
        } else if (
            jobId == keccak256(abi.encodePacked("uint128")) ||
            jobId == keccak256(abi.encodePacked("int256")) ||
            jobId == keccak256(abi.encodePacked("bool"))
        ) {
            uint8 maxSubmission = uint8(sOracles.length / 2);
            if (numSubmission != 1 && numSubmission > maxSubmission) {
                revert InvalidNumSubmission();
            }
        }
    }

    function computeRequestId(
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256) {
        return uint256(keccak256(abi.encode(sender, accId, nonce)));
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) public view returns (bool) {
        uint256 oraclesLength = sOracles.length;
        for (uint256 i = 0; i < oraclesLength; ++i) {
            uint256 requestId = computeRequestId(consumer, accId, nonce);
            if (isValidRequestId(requestId)) {
                return true;
            }
        }
        return false;
    }

    function requestData(
        Orakl.Request memory req,
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        bool isDirectPayment
    ) private returns (uint256) {
        validateNumSubmission(req.id, numSubmission);

        if (!sPrepayment.isValidAccount(accId, msg.sender)) {
            revert InvalidConsumer(accId, msg.sender);
        }

        if (callbackGasLimit > sConfig.maxGasLimit) {
            revert GasLimitTooBig(callbackGasLimit, sConfig.maxGasLimit);
        }

        uint64 nonce = sPrepayment.increaseNonce(accId, msg.sender);

        uint256 requestId = computeRequestId(msg.sender, accId, nonce);
        sRequestIdToCommitment[requestId] = computeCommitment(
            requestId,
            block.number,
            accId,
            numSubmission,
            callbackGasLimit,
            msg.sender,
            isDirectPayment,
            req.id
        );

        sRequestOwner[requestId] = msg.sender;

        emit DataRequested(
            requestId,
            req.id,
            accId,
            callbackGasLimit,
            msg.sender,
            isDirectPayment,
            numSubmission,
            req.buf.buf
        );

        return requestId;
    }

    function validateDataResponse(RequestCommitment memory rc, uint256 requestId) private view {
        if (!sIsOracleRegistered[msg.sender]) {
            revert UnregisteredOracleFulfillment(msg.sender);
        }

        if (sSubmission[requestId].submitted[msg.sender]) {
            revert OracleAlreadySubmitted();
        }

        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (
            commitment !=
            computeCommitment(
                requestId,
                rc.blockNum,
                rc.accId,
                rc.numSubmission,
                rc.callbackGasLimit,
                rc.sender,
                rc.isDirectPayment,
                rc.jobId
            )
        ) {
            revert IncorrectCommitment();
        }
    }

    function fulfill(bytes memory resp, RequestCommitment memory rc) private returns (bool) {
        // Call with explicitly the amount of callback gas requested
        // Important to not let them exhaust the gas budget and avoid oracle payment.
        // Do not allow any non-view/non-pure coordinator functions to be called
        // during the consumers callback code via reentrancyLock.
        // Note that callWithExactGas will revert if we do not have sufficient gas
        // to give the callee their requested amount.
        sConfig.reentrancyLock = true;
        bool success = callWithExactGas(rc.callbackGasLimit, rc.sender, resp);
        sConfig.reentrancyLock = false;
        return success;
    }

    function pay(
        RequestCommitment memory rc,
        uint256 startGas,
        address[] memory oracles
    ) private returns (uint256) {
        uint256 oraclesLength = oracles.length;

        if (rc.isDirectPayment) {
            // [temporary] account
            (uint256 totalFee, uint256 operatorFee) = sPrepayment.chargeFeeTemporary(rc.accId);

            if (operatorFee > 0) {
                uint256 paid;
                uint256 feePerOperator = operatorFee / oraclesLength;

                for (uint8 i = 0; i < oraclesLength - 1; ++i) {
                    sPrepayment.chargeOperatorFeeTemporary(feePerOperator, oracles[i]);
                    paid += feePerOperator;
                }

                sPrepayment.chargeOperatorFeeTemporary(
                    operatorFee - paid,
                    oracles[oraclesLength - 1]
                );
            }

            return totalFee;
        } else {
            // [regular] account

            uint256 serviceFee = serviceFeeByAcc(rc.accId, rc.numSubmission);
            if (serviceFee > 0) {
                uint256 operatorFee = sPrepayment.chargeFee(rc.accId, serviceFee);
                uint256 feePerOperator = operatorFee / oraclesLength;
                uint256 paid;
                for (uint256 i = 0; i < oraclesLength - 1; ++i) {
                    sPrepayment.chargeOperatorFee(rc.accId, feePerOperator, oracles[i]);
                    paid += feePerOperator;
                }
                uint256 gasFee = calculateGasCost(startGas);
                sPrepayment.chargeOperatorFee(
                    rc.accId,
                    (operatorFee - paid) + gasFee,
                    oracles[oraclesLength - 1]
                );
                return gasFee + serviceFee;
            } else return 0;
        }
    }

    function cleanupAfterFulfillment(uint256 requestId) private returns (address[] memory) {
        address[] memory oracles = sSubmission[requestId].oracles;

        for (uint8 i = 0; i < oracles.length; ++i) {
            delete sSubmission[requestId].submitted[oracles[i]];
        }

        delete sSubmission[requestId];
        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        return oracles;
    }

    /**
     * @notice Loss-less conversion of array items from uint128 to int256.
     * @dev uint128: 0     - 2^128-1
     * @dev int256:  2^128 - 2^128-1
     * @param arr - array of uint128 values
     * @return array of int256 values
     */
    function uint128ToInt256(uint128[] memory arr) private pure returns (int256[] memory) {
        int256[] memory responses = new int256[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            responses[i] = int256(uint256(arr[i]));
        }
        return responses;
    }

    function computeCommitment(
        uint256 requestId,
        uint256 blockNumber,
        uint64 accId,
        uint8 numSubmission,
        uint32 callbackGasLimit,
        address sender,
        bool isDirectPayment,
        bytes32 jobId
    ) private pure returns (bytes32) {
        return
            keccak256(
                abi.encode(
                    requestId,
                    blockNumber,
                    accId,
                    callbackGasLimit,
                    numSubmission,
                    sender,
                    isDirectPayment,
                    jobId
                )
            );
    }
}
