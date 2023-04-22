// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IRequestResponseCoordinator.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./RequestResponseConsumerBase.sol";
import "./RequestResponseConsumerFulfill.sol";
import "./libraries/Orakl.sol";
import "./libraries/Median.sol";
import "./libraries/MajorityVoting.sol";

contract RequestResponseCoordinator is
    Ownable,
    ICoordinatorBase,
    IRequestResponseCoordinator,
    ITypeAndVersion
{
    using Orakl for Orakl.Request;

    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    uint8 public constant MAX_ORACLES = 255;

    address[] public sOracles;
    mapping(address => bool) private sIsOracleRegistered;

    /* requestID */
    /* commitment */
    mapping(uint256 => bytes32) private sRequestIdToCommitment;

    /* requestID */
    /* owner */
    mapping(uint256 => address) private sRequestOwner;

    uint256 public sMinBalance;

    IPrepayment private sPrepayment;

    struct Config {
        uint32 maxGasLimit;
        // Reentrancy protection.
        bool reentrancyLock;
        // Gas to cover oracle payment after we calculate the payment.
        // We make it configurable in case those operations are repriced.
        uint32 gasAfterPaymentCalculation;
    }
    Config private sConfig;

    struct FeeConfig {
        // Flat fee charged per fulfillment in millionths of KLAY
        // So fee range is [0, 2^32/10^6].
        uint32 fulfillmentFlatFeeKlayPPMTier1;
        uint32 fulfillmentFlatFeeKlayPPMTier2;
        uint32 fulfillmentFlatFeeKlayPPMTier3;
        uint32 fulfillmentFlatFeeKlayPPMTier4;
        uint32 fulfillmentFlatFeeKlayPPMTier5;
        uint24 reqsForTier2;
        uint24 reqsForTier3;
        uint24 reqsForTier4;
        uint24 reqsForTier5;
    }
    FeeConfig private sFeeConfig;

    struct DirectPaymentConfig {
        uint256 fulfillmentFee;
        uint256 baseFee;
    }

    DirectPaymentConfig private sDirectPaymentConfig;
    mapping(bytes32 => bool) sJobId;

    /* request ID */
    /* number of requested submissions */
    mapping(uint256 => uint8) private sRequestToNumSubmission;

    /* request ID */
    /* oracle submission participants */
    mapping(uint256 => address[]) private sRequestToOracles;

    mapping(uint256 => int256[]) private sRequestToSubmissionInt256;
    mapping(uint256 => uint256[]) private sRequestToSubmissionUint256;
    mapping(uint256 => bool[]) private sRequestToSubmissionBool;

    error TooManyOracles();
    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error UnregisteredOracleFulfillment(address oracle);
    error NoCorrespondingRequest();
    error IncorrectCommitment();
    error NotRequestOwner();
    error Reentrant();
    error InsufficientPayment(uint256 have, uint256 want);
    error RefundFailure();
    error GasLimitTooBig(uint32 have, uint32 want);
    error OracleAlreadyRegistered(address oracle);
    error NoSuchOracle(address oracle);
    error InvalidJobId();
    error InvalidNumSubmission();

    event DataRequested(
        uint256 indexed requestId,
        bytes32 jobId,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        address indexed sender,
        bool isDirectPayment,
        bytes data
    );

    event DataRequestCancelled(uint256 indexed requestId);
    event ConfigSet(uint32 maxGasLimit, uint32 gasAfterPaymentCalculation, FeeConfig feeConfig);
    event DirectPaymentConfigSet(uint256 fulfillmentFee, uint256 baseFee);

    event OracleRegistered(address oracle);
    event OracleDeregistered(address oracle);
    event MinBalanceSet(uint256 minBalance);
    event PrepaymentSet(address prepayment);

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
    event DataSubmitted(address oracle, uint256 requestId, bool success);

    modifier nonReentrant() {
        if (sConfig.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

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
     * @notice Sets the general configuration
     * @param maxGasLimit global max for request gas limit
     * @param gasAfterPaymentCalculation gas used in doing accounting after completing the gas measurement
     * @param feeConfig fee tier configuration
     */
    function setConfig(
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig memory feeConfig
    ) external onlyOwner {
        sConfig = Config({
            maxGasLimit: maxGasLimit,
            gasAfterPaymentCalculation: gasAfterPaymentCalculation,
            reentrancyLock: false
        });
        sFeeConfig = feeConfig;
        emit ConfigSet(maxGasLimit, gasAfterPaymentCalculation, sFeeConfig);
    }

    function getConfig()
        external
        view
        returns (uint32 maxGasLimit, uint32 gasAfterPaymentCalculation)
    {
        return (sConfig.maxGasLimit, sConfig.gasAfterPaymentCalculation);
    }

    function getFeeConfig()
        external
        view
        returns (
            uint32 fulfillmentFlatFeeKlayPPMTier1,
            uint32 fulfillmentFlatFeeKlayPPMTier2,
            uint32 fulfillmentFlatFeeKlayPPMTier3,
            uint32 fulfillmentFlatFeeKlayPPMTier4,
            uint32 fulfillmentFlatFeeKlayPPMTier5,
            uint24 reqsForTier2,
            uint24 reqsForTier3,
            uint24 reqsForTier4,
            uint24 reqsForTier5
        )
    {
        return (
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier1,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier2,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier3,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier4,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier5,
            sFeeConfig.reqsForTier2,
            sFeeConfig.reqsForTier3,
            sFeeConfig.reqsForTier4,
            sFeeConfig.reqsForTier5
        );
    }

    function setDirectPaymentConfig(
        DirectPaymentConfig memory directPaymentConfig
    ) external onlyOwner {
        sDirectPaymentConfig = directPaymentConfig;
        emit DirectPaymentConfigSet(
            directPaymentConfig.fulfillmentFee,
            directPaymentConfig.baseFee
        );
    }

    function getDirectPaymentConfig() external view returns (uint256, uint256) {
        return (sDirectPaymentConfig.fulfillmentFee, sDirectPaymentConfig.baseFee);
    }

    function getPrepaymentAddress() external view returns (address) {
        return address(sPrepayment);
    }

    function setMinBalance(uint256 minBalance) external onlyOwner {
        sMinBalance = minBalance;
        emit MinBalanceSet(minBalance);
    }

    function validateNumSubmission(bytes32 jobId, uint8 numSubmission) internal view {
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

    // TODO description
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numSubmission
    ) external nonReentrant returns (uint256) {
        if (!sJobId[req.id]) {
            revert InvalidJobId();
        }
        validateNumSubmission(req.id, numSubmission);

        uint256 balance = sPrepayment.getBalance(accId);
        if (balance < sMinBalance) {
            revert InsufficientPayment(balance, sMinBalance);
        }

        bool isDirectPayment = false;
        uint256 requestId = requestDataInternal(
            req,
            accId,
            callbackGasLimit,
            numSubmission,
            isDirectPayment
        );

        return requestId;
    }

    // TODO description
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) external payable returns (uint256) {
        if (!sJobId[req.id]) {
            revert InvalidJobId();
        }
        validateNumSubmission(req.id, numSubmission);

        uint256 fee = estimateDirectPaymentFee();
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = sPrepayment.createTemporaryAccount();
        bool isDirectPayment = true;
        uint256 requestId = requestDataInternal(
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

    /**
     * @inheritdoc IRequestResponseCoordinator
     */
    function cancelRequest(uint256 requestId) external {
        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (sRequestOwner[requestId] != msg.sender) {
            revert NotRequestOwner();
        }

        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        emit DataRequestCancelled(requestId);
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

    function estimateDirectPaymentFee() internal view returns (uint256) {
        return sDirectPaymentConfig.fulfillmentFee + sDirectPaymentConfig.baseFee;
    }

    function requestDataInternal(
        Orakl.Request memory req,
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        bool isDirectPayment
    ) internal returns (uint256) {
        if (!sPrepayment.isValid(accId, msg.sender)) {
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
        sRequestIdToCommitment[requestId] = keccak256(
            abi.encode(requestId, block.number, accId, callbackGasLimit, msg.sender)
        );

        sRequestOwner[requestId] = msg.sender;
        sRequestToNumSubmission[requestId] = numSubmission;

        emit DataRequested(
            requestId,
            req.id,
            accId,
            callbackGasLimit,
            msg.sender,
            isDirectPayment,
            req.buf.buf
        );

        return requestId;
    }

    function calculatePaymentAmount(
        uint64 accId,
        uint256 startGas
    ) internal view returns (uint256) {
        uint64 reqCount = sPrepayment.getReqCount(accId);
        uint32 fulfillmentFlatFeeKlayPPM = getFeeTier(reqCount);

        uint256 paymentNoFee = tx.gasprice *
            (sConfig.gasAfterPaymentCalculation + startGas - gasleft());
        uint256 fee = 1e12 * uint256(fulfillmentFlatFeeKlayPPM);

        return paymentNoFee + fee;
    }

    function calculatePaymentNoFee(uint256 startGas) internal view returns (uint256) {
        return tx.gasprice * (sConfig.gasAfterPaymentCalculation + startGas - gasleft());
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
            uint256 reqId = computeRequestId(consumer, accId, nonce);
            if (sRequestIdToCommitment[reqId] != 0) {
                return true;
            }
        }
        return false;
    }

    /*
     * @notice Compute fee based on the request count
     * @param reqCount number of requests
     * @return feePPM fee in KLAY PPM
     */
    function getFeeTier(uint64 reqCount) public view returns (uint32) {
        FeeConfig memory fc = sFeeConfig;
        if (0 <= reqCount && reqCount <= fc.reqsForTier2) {
            return fc.fulfillmentFlatFeeKlayPPMTier1;
        }
        if (fc.reqsForTier2 < reqCount && reqCount <= fc.reqsForTier3) {
            return fc.fulfillmentFlatFeeKlayPPMTier2;
        }
        if (fc.reqsForTier3 < reqCount && reqCount <= fc.reqsForTier4) {
            return fc.fulfillmentFlatFeeKlayPPMTier3;
        }
        if (fc.reqsForTier4 < reqCount && reqCount <= fc.reqsForTier5) {
            return fc.fulfillmentFlatFeeKlayPPMTier4;
        }
        return fc.fulfillmentFlatFeeKlayPPMTier5;
    }

    function computeRequestId(
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256) {
        return uint256(keccak256(abi.encode(sender, accId, nonce)));
    }

    /**
     * @dev calls target address with exactly gasAmount gas and data as calldata
     * or reverts if at least gasAmount gas is not available.
     */
    function callWithExactGas(
        uint256 gasAmount,
        address target,
        bytes memory data
    ) private returns (bool success) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            let g := gas()
            // Compute g -= GAS_FOR_CALL_EXACT_CHECK and check for underflow
            // The gas actually passed to the callee is min(gasAmount, 63//64*gas available).
            // We want to ensure that we revert if gasAmount >  63//64*gas available
            // as we do not want to provide them with less, however that check itself costs
            // gas.  GAS_FOR_CALL_EXACT_CHECK ensures we have at least enough gas to be able
            // to revert if gasAmount >  63//64*gas available.
            if lt(g, GAS_FOR_CALL_EXACT_CHECK) {
                revert(0, 0)
            }
            g := sub(g, GAS_FOR_CALL_EXACT_CHECK)
            // if g - g//64 <= gasAmount, revert
            // (we subtract g//64 because of EIP-150)
            if iszero(gt(sub(g, div(g, 64)), gasAmount)) {
                revert(0, 0)
            }
            // solidity calls check that a contract actually exists at the destination, so we do the same
            if iszero(extcodesize(target)) {
                revert(0, 0)
            }
            // call and return whether we succeeded. ignore return data
            // call(gas,addr,value,argsOffset,argsLength,retOffset,retLength)
            success := call(gasAmount, target, 0, add(data, 0x20), mload(data), 0, 0)
        }
        return success;
    }

    function validateDataResponse(RequestCommitment memory rc, uint256 requestId) internal view {
        if (!sIsOracleRegistered[msg.sender]) {
            revert UnregisteredOracleFulfillment(msg.sender);
        }

        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (
            commitment !=
            keccak256(abi.encode(requestId, rc.blockNum, rc.accId, rc.callbackGasLimit, rc.sender))
        ) {
            revert IncorrectCommitment();
        }
    }

    function fulfill(bytes memory resp, RequestCommitment memory rc) internal returns (bool) {
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
    ) internal returns (uint256) {
        if (isDirectPayment) {
            (uint256 totalAmount, uint256 operatorFee) = sPrepayment.chargeFeeTemporary(rc.accId);

            uint256 paymentNoFee = calculatePaymentNoFee(startGas);

            uint8 oraclesLength = uint8(oracles.length);
            uint256 amountForEachOperator = 0;
            if (operatorFee > 0) {
                amountForEachOperator = operatorFee;
            }
            if (operatorFee > paymentNoFee) {
                amountForEachOperator = (operatorFee - paymentNoFee) / oraclesLength;
            }
            if (amountForEachOperator > 0) {
                for (uint8 i = 0; i < oraclesLength - 1; ++i) {
                    sPrepayment.chargeOperatorFeeTemporary(amountForEachOperator, oracles[i]);
                }
            }

            sPrepayment.chargeOperatorFeeTemporary(
                amountForEachOperator + paymentNoFee,
                oracles[oraclesLength - 1]
            );
            sPrepayment.increaseReqCountTemporary(rc.accId);

            return totalAmount;
        } else {
            uint256 payment = calculatePaymentAmount(rc.accId, startGas);
            sPrepayment.chargeFee(rc.accId, payment);
            uint8 burnFeeRatio = sPrepayment.getBurnFeeRatio();
            uint8 protocolFeeRatio = sPrepayment.getProtocolFeeRatio();
            uint256 burnFee = (burnFeeRatio * payment) / 100;
            uint256 protocolFee = (protocolFeeRatio * payment) / 100;

            uint256 operatorFee = payment - burnFee - protocolFee;
            uint256 paymentNoFee = calculatePaymentNoFee(startGas);
            uint256 amountForEachOperator = 0;
            if (operatorFee > 0) {
                amountForEachOperator = operatorFee;
            }
            if (operatorFee > paymentNoFee) {
                amountForEachOperator = (operatorFee - paymentNoFee) / oracles.length;
            }
            for (uint256 i = 0; i < oracles.length - 1; i++) {
                sPrepayment.payOperatorFee(rc.accId, amountForEachOperator, oracles[i], burnFee);
            }
            sPrepayment.payOperatorFee(
                rc.accId,
                amountForEachOperator + paymentNoFee,
                oracles[oracles.length - 1],
                burnFee
            );
            sPrepayment.increaseNonce(rc.accId, msg.sender);
            sPrepayment.increaseReqCount(rc.accId);

            return payment;
        }
    }

    function cleanupAfterFulfillment(uint256 requestId) internal {
        delete sRequestToNumSubmission[requestId];
        delete sRequestToOracles[requestId];
        delete sRequestIdToCommitment[requestId];
    }

    // FIXME wrong
    function arrUintToInt(uint256[] memory arr) internal pure returns (int256[] memory) {
        int256[] memory responses = new int256[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            responses[i] = int256(arr[i]);
        }
        return responses;
    }

    function fulfillDataRequestUint256(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        uint256[] storage arrRes = sRequestToSubmissionUint256[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
        oracles.push(msg.sender);

        if (arrRes.length < sRequestToNumSubmission[requestId]) {
            emit DataSubmitted(msg.sender, requestId, true);
            return 0;
        }

        int256[] memory responses = arrUintToInt(arrRes);
        uint256 aggregatedResponse = uint256(Median.calculate(responses));

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillUint256.rawFulfillDataRequestUint256.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionUint256[requestId];

        emit DataRequestFulfilledUint256(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        int256[] storage arrRes = sRequestToSubmissionInt256[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
        oracles.push(msg.sender);

        if (arrRes.length < sRequestToNumSubmission[requestId]) {
            emit DataSubmitted(msg.sender, requestId, true);
            return 0;
        }

        int256 aggregatedResponse = Median.calculate(arrRes);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillInt256.rawFulfillDataRequestInt256.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionInt256[requestId];

        emit DataRequestFulfilledInt256(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bool[] storage arrRes = sRequestToSubmissionBool[requestId];
        address[] storage oracles = sRequestToOracles[requestId];
        arrRes.push(response);
        oracles.push(msg.sender);

        if (arrRes.length < sRequestToNumSubmission[requestId]) {
            emit DataSubmitted(msg.sender, requestId, true);
            return 0;
        }

        bool aggregatedResponse = MajorityVoting.voting(arrRes);
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBool.rawFulfillDataRequestBool.selector,
            requestId,
            aggregatedResponse
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);
        delete sRequestToSubmissionBool[requestId];

        emit DataRequestFulfilledBool(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillString.rawFulfillDataRequestString.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

        emit DataRequestFulfilledString(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes32.rawFulfillDataRequestBytes32.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

        emit DataRequestFulfilledBytes32(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);

        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes.rawFulfillDataRequestBytes.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);

        address[] storage oracles = sRequestToOracles[requestId];
        oracles.push(msg.sender);
        uint256 payment = pay(rc, isDirectPayment, startGas, oracles);

        cleanupAfterFulfillment(requestId);

        emit DataRequestFulfilledBytes(requestId, response, payment, success);
        return payment;
    }
}
