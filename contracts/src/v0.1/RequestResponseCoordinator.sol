// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./interfaces/IRequestResponseCoordinatorBase.sol";
import "./RequestResponseConsumerBase.sol";
import "./RequestResponseConsumerFulfill.sol";
import "./CoordinatorBase.sol";
import "./libraries/Orakl.sol";
import "./libraries/Median.sol";
import "./libraries/MajorityVoting.sol";

contract RequestResponseCoordinator is
    CoordinatorBase,
    IRequestResponseCoordinatorBase,
    ITypeAndVersion
{
    uint8 public constant MAX_ORACLES = 255;

    using Orakl for Orakl.Request;

    /* oracle */
    /* registration status */
    mapping(address => bool) private sIsOracleRegistered;

    /* jobId */
    /* ability to request for the job */
    mapping(bytes32 => bool) private sJobId;

    /* request ID */
    /* oracle submission participants */
    mapping(uint256 => address[]) private sRequestToOracles;

    mapping(uint256 => int256[]) private sRequestToSubmissionInt256;
    mapping(uint256 => uint256[]) private sRequestToSubmissionUint256;
    mapping(uint256 => bool[]) private sRequestToSubmissionBool;

    error TooManyOracles();
    error UnregisteredOracleFulfillment(address oracle);
    error InvalidJobId();
    error InvalidNumSubmission();

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
    event DataRequestFulfilledUint256(
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
        sJobId[keccak256(abi.encodePacked("uint256"))] = true;
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
        for (uint256 i; i < oraclesLength; ++i) {
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
        if (!sJobId[req.id]) {
            // move to internal
            revert InvalidJobId();
        }
        validateNumSubmission(req.id, numSubmission);

        (uint256 balance, uint64 reqCount, , ) = sPrepayment.getAccount(accId);
        uint256 minBalance = estimateTotalFee(reqCount, numSubmission, callbackGasLimit);
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
        uint8 numSubmission
    ) external payable returns (uint256) {
        if (!sJobId[req.id]) {
            revert InvalidJobId();
        }
        validateNumSubmission(req.id, numSubmission);

        uint64 reqCount = 0;
        uint256 fee = estimateTotalFee(reqCount, numSubmission, callbackGasLimit);
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = sPrepayment.createTemporaryAccount();
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
            (bool sent, ) = msg.sender.call{value: remaining}("");
            if (!sent) {
                revert RefundFailure();
            }
        }

        return requestId;
    }

    function fulfillDataRequestUint256(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        uint256[] storage arrRes = sRequestToSubmissionUint256[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
        oracles.push(msg.sender);

        if (arrRes.length < rc.numSubmission) {
            emit DataSubmitted(msg.sender, requestId);
            return;
        }

        int256[] memory responses = arrUintToInt(arrRes);
        uint256 aggregatedResponse = uint256(Median.calculate(responses));

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillUint256.rawFulfillDataRequest.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionUint256[requestId];

        emit DataRequestFulfilledUint256(requestId, response, payment, success);
    }

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        int256[] storage arrRes = sRequestToSubmissionInt256[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
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
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionInt256[requestId];

        emit DataRequestFulfilledInt256(requestId, response, payment, success);
    }

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bool[] storage arrRes = sRequestToSubmissionBool[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
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
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionBool[requestId];

        emit DataRequestFulfilledBool(requestId, response, payment, success);
    }

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillString.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

        emit DataRequestFulfilledString(requestId, response, payment, success);
    }

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes32.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

        emit DataRequestFulfilledBytes32(requestId, response, payment, success);
    }

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

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
        for (uint256 i; i < oraclesLength; ++i) {
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
        if (!sPrepayment.isValidAccount(accId, msg.sender)) {
            revert InvalidConsumer(accId, msg.sender);
        }

        // TODO update comment
        // No lower bound on the requested gas limit. A user could request 0
        // and they would simply be billed for the proof verification and wouldn't be
        // able to do anything with the random value.
        if (callbackGasLimit > sConfig.maxGasLimit) {
            revert GasLimitTooBig(callbackGasLimit, sConfig.maxGasLimit);
        }

        uint64 nonce = sPrepayment.increaseNonce(accId, msg.sender);

        uint256 requestId = computeRequestId(msg.sender, accId, nonce);
        sRequestIdToCommitment[requestId] = computeCommitment(
            requestId,
            block.number,
            accId,
            callbackGasLimit,
            numSubmission,
            msg.sender
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
                rc.callbackGasLimit,
                rc.numSubmission,
                rc.sender
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
        bool isDirectPayment,
        uint256 startGas,
        address[] memory oracles
    ) private returns (uint256) {
        uint256 oraclesLength = oracles.length;

        if (isDirectPayment) {
            // [temporary] account
            (uint256 totalFee, uint256 operatorFee) = sPrepayment.chargeFeeTemporary(rc.accId);

            if (operatorFee > 0) {
                uint256 paid;
                uint256 feePerOperator = operatorFee / oraclesLength;

                for (uint8 i; i < oraclesLength - 1; ++i) {
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
            uint64 reqCount = sPrepayment.getReqCount(rc.accId);
            uint256 serviceFee = calculateServiceFee(reqCount) * rc.numSubmission;
            uint256 operatorFee = sPrepayment.chargeFee(rc.accId, serviceFee);
            uint256 feePerOperator = operatorFee / oraclesLength;

            uint256 paid;
            for (uint256 i; i < oraclesLength - 1; ++i) {
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
        }
    }

    function cleanupAfterFulfillment(uint256 requestId) private {
        delete sRequestToOracles[requestId];
        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];
    }

    // FIXME wrong
    function arrUintToInt(uint256[] memory arr) private pure returns (int256[] memory) {
        int256[] memory responses = new int256[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            responses[i] = int256(arr[i]);
        }
        return responses;
    }

    function computeCommitment(
        uint256 requestId,
        uint256 blockNumber,
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender
    ) private pure returns (bytes32) {
        return
            keccak256(
                abi.encode(requestId, blockNumber, accId, callbackGasLimit, numSubmission, sender)
            );
    }

    function validateNumSubmission(bytes32 jobId, uint8 numSubmission) private view {
        if (numSubmission == 0) {
            revert InvalidNumSubmission();
        } else if (jobId == keccak256(abi.encodePacked("bool")) && numSubmission % 2 == 0) {
            revert InvalidNumSubmission();
        } else if (
            jobId == keccak256(abi.encodePacked("uint256")) ||
            jobId == keccak256(abi.encodePacked("int256")) ||
            jobId == keccak256(abi.encodePacked("bool"))
        ) {
            uint8 maxSubmission = uint8(sOracles.length / 2);
            if (numSubmission != 1 && numSubmission > maxSubmission) {
                revert InvalidNumSubmission();
            }
        }
    }
}
