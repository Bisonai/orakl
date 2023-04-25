// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/VRFCoordinatorV2.sol

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./interfaces/IVRFCoordinator.sol";
import "./libraries/VRF.sol";
import "./VRFConsumerBase.sol";

contract VRFCoordinator is Ownable, ICoordinatorBase, ITypeAndVersion, IVRFCoordinator {
    uint32 public constant MAX_NUM_WORDS = 500;
    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    address[] public sOracles;
    mapping(address => bytes32) private sOracleToKeyHash;
    bytes32[] public sKeyHashes;
    mapping(bytes32 => address) private sKeyHashToOracle;
    mapping(uint256 => bytes32) private sRequestIdToCommitment;

    /* requestID */
    /* owner */
    mapping(uint256 => address) private sRequestOwner;

    uint256 public sMinBalance;

    // RequestCommitment holds information sent from off-chain oracle
    // describing details of request.
    struct RequestCommitment {
        uint64 blockNum;
        uint64 accId;
        uint32 callbackGasLimit;
        uint32 numWords;
        address sender;
    }

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

    IPrepayment private sPrepayment;

    struct DirectPaymentConfig {
        uint256 fulfillmentFee;
        uint256 baseFee;
    }

    DirectPaymentConfig private sDirectPaymentConfig;

    error InvalidKeyHash(bytes32 keyHash);
    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error GasLimitTooBig(uint32 have, uint32 want);
    error NumWordsTooBig(uint32 have, uint32 want);
    error OracleAlreadyRegistered(address oracle);
    error ProvingKeyAlreadyRegistered(bytes32 keyHash);
    error NoSuchOracle(address oracle);
    error NoSuchProvingKey(bytes32 keyHash);
    error NoCorrespondingRequest();
    error IncorrectCommitment();
    error NotRequestOwner();
    error Reentrant();
    error InsufficientPayment(uint256 have, uint256 want);
    error RefundFailure();

    event RandomWordsRequestCanceled(uint256 indexed requestId);
    event OracleRegistered(address indexed oracle, bytes32 keyHash);
    event OracleDeregistered(address indexed oracle, bytes32 keyHash);
    event RandomWordsRequested(
        bytes32 indexed keyHash,
        uint256 requestId,
        uint256 preSeed,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        uint32 numWords,
        address indexed sender,
        bool isDirectPayment
    );
    event RandomWordsFulfilled(
        uint256 indexed requestId,
        uint256 outputSeed,
        uint256 payment,
        bool success
    );
    event ConfigSet(uint32 maxGasLimit, uint32 gasAfterPaymentCalculation, FeeConfig feeConfig);
    event DirectPaymentConfigSet(uint256 fulfillmentFee, uint256 baseFee);
    event MinBalanceSet(uint256 minBalance);
    event PrepaymentSet(address prepayment);

    modifier nonReentrant() {
        if (sConfig.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    modifier onlyValidKeyHash(bytes32 keyHash) {
        if (sKeyHashToOracle[keyHash] == address(0)) {
            revert InvalidKeyHash(keyHash);
        }
        _;
    }

    constructor(address prepayment) {
        sPrepayment = IPrepayment(prepayment);
        emit PrepaymentSet(prepayment);
    }

    /**
     * @notice Registers an oracle and its proving key.
     * @param oracle address of the oracle
     * @param publicProvingKey key that oracle can use to submit VRF fulfillments
     */
    function registerOracle(
        address oracle,
        uint256[2] calldata publicProvingKey
    ) external onlyOwner {
        if (sOracleToKeyHash[oracle] != bytes32(0)) {
            revert OracleAlreadyRegistered(oracle);
        }

        bytes32 kh = hashOfKey(publicProvingKey);
        if (sKeyHashToOracle[kh] != address(0)) {
            revert ProvingKeyAlreadyRegistered(kh);
        }

        sOracles.push(oracle);
        sKeyHashes.push(kh);
        sOracleToKeyHash[oracle] = kh;
        sKeyHashToOracle[kh] = oracle;

        emit OracleRegistered(oracle, kh);
    }

    /**
     * @notice Deregisters an oracle.
     * @param oracle address representing oracle that can submit VRF fulfillments
     */
    function deregisterOracle(address oracle) external onlyOwner {
        bytes32 kh = sOracleToKeyHash[oracle];
        if (kh == bytes32(0) || sKeyHashToOracle[kh] == address(0)) {
            revert NoSuchOracle(oracle);
        }
        delete sOracleToKeyHash[oracle];
        delete sKeyHashToOracle[kh];

        uint256 oraclesLength = sOracles.length;
        for (uint256 i = 0; i < oraclesLength; ++i) {
            if (sOracles[i] == oracle) {
                // oracles
                address lastOracle = sOracles[oraclesLength - 1];
                sOracles[i] = lastOracle;
                sOracles.pop();

                // key hashes
                bytes32 lastKeyHash = sKeyHashes[oraclesLength - 1];
                sKeyHashes[i] = lastKeyHash;
                sKeyHashes.pop();
                break;
            }
        }

        emit OracleDeregistered(oracle, kh);
    }

    /**
     * @notice Sets the configuration of the VRF coordinator
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

    /**
     * @inheritdoc IVRFCoordinator
     */
    function getRequestConfig() external view returns (uint32, bytes32[] memory) {
        return (sConfig.maxGasLimit, sKeyHashes);
    }

    function getDirectPaymentConfig() external view returns (uint256, uint256) {
        return (sDirectPaymentConfig.fulfillmentFee, sDirectPaymentConfig.baseFee);
    }

    /**
     * @notice Get request commitment
     * @param requestId id of request
     * @dev used to determine if a request is fulfilled or not
     */
    function getCommitment(uint256 requestId) external view returns (bytes32) {
        return sRequestIdToCommitment[requestId];
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

    function getPrepaymentAddress() external view returns (address) {
        return address(sPrepayment);
    }

    function setMinBalance(uint256 minBalance) external onlyOwner {
        sMinBalance = minBalance;
        emit MinBalanceSet(minBalance);
    }

    /**
     * @inheritdoc IVRFCoordinator
     */
    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external nonReentrant onlyValidKeyHash(keyHash) returns (uint256) {
        // TODO check if he is one of the consumers

        uint256 balance = sPrepayment.getBalance(accId);
        if (balance < sMinBalance) {
            revert InsufficientPayment(balance, sMinBalance);
        }

        bool isDirectPayment = false;
        uint256 requestId = requestRandomWordsInternal(
            keyHash,
            accId,
            callbackGasLimit,
            numWords,
            isDirectPayment
        );

        return requestId;
    }

    /**
     * @inheritdoc IVRFCoordinator
     */
    function requestRandomWords(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external payable nonReentrant onlyValidKeyHash(keyHash) returns (uint256) {
        uint256 fee = estimateDirectPaymentFee();
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = sPrepayment.createTemporaryAccount();

        bool isDirectPayment = true;
        uint256 requestId = requestRandomWordsInternal(
            keyHash,
            accId,
            callbackGasLimit,
            numWords,
            isDirectPayment
        );
        sPrepayment.depositTemporary{value: fee}(accId);

        uint256 remaining = msg.value - fee;
        if (remaining > 0) {
            (bool sent, ) = msg.sender.call{value: remaining}("");
            if (!sent) {
                revert RefundFailure();
            }
        }

        return requestId;
    }

    /*
     * @notice Fulfill a randomness request
     * @param proof contains the proof and randomness
     * @param rc request commitment pre-image, committed to at request time
     * @return payment amount billed to the account
     * @dev simulated offchain to determine if sufficient balance is present to fulfill the request
     */
    function fulfillRandomWords(
        VRF.Proof memory proof,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        (bytes32 keyHash, uint256 requestId, uint256 randomness) = getRandomnessFromProof(
            proof,
            rc
        );

        uint256[] memory randomWords = new uint256[](rc.numWords);
        for (uint256 i = 0; i < rc.numWords; i++) {
            randomWords[i] = uint256(keccak256(abi.encode(randomness, i)));
        }

        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        VRFConsumerBase v;
        bytes memory resp = abi.encodeWithSelector(
            v.rawFulfillRandomWords.selector,
            requestId,
            randomWords
        );

        // Call with explicitly the amount of callback gas requested
        // Important to not let them exhaust the gas budget and avoid oracle payment.
        // Do not allow any non-view/non-pure coordinator functions to be called
        // during the consumers callback code via reentrancyLock.
        // Note that callWithExactGas will revert if we do not have sufficient gas
        // to give the callee their requested amount.
        sConfig.reentrancyLock = true;
        bool success = callWithExactGas(rc.callbackGasLimit, rc.sender, resp);
        sConfig.reentrancyLock = false;

        uint256 payment = pay(rc, isDirectPayment, startGas, keyHash);
        emit RandomWordsFulfilled(requestId, randomness, payment, success);
        return payment;
    }

    function pay(
        RequestCommitment memory rc,
        bool isDirectPayment,
        uint256 startGas,
        bytes32 keyHash
    ) internal returns (uint256) {
        if (isDirectPayment) {
            (uint256 totalFee, uint256 operatorFee) = sPrepayment.chargeFeeTemporary(rc.accId);
            if (operatorFee > 0) {
                sPrepayment.chargeOperatorFeeTemporary(operatorFee, sKeyHashToOracle[keyHash]);
            }

            sPrepayment.increaseReqCountTemporary(rc.accId);
            return totalFee;
        } else {
            uint256 serviceFee = calculateFee(rc.accId);
            uint256 gasFee = calculateGasCost(startGas);
            uint256 operatorFee = sPrepayment.chargeFee(rc.accId, serviceFee);

            if (operatorFee > 0) {
                sPrepayment.chargeOperatorFee(rc.accId, operatorFee, sKeyHashToOracle[keyHash]);
            }

            sPrepayment.increaseReqCount(rc.accId);
            return gasFee + serviceFee;
        }
    }

    /**
     * @inheritdoc IVRFCoordinator
     */
    function cancelRequest(uint256 requestId) external {
        if (!isValidRequestId(requestId)) {
            revert NoCorrespondingRequest();
        }

        if (sRequestOwner[requestId] != msg.sender) {
            revert NotRequestOwner();
        }

        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        emit RandomWordsRequestCanceled(requestId);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "VRFCoordinator v0.1";
    }

    /**
     * @notice Find key hash associated with given oracle address.
     * @return keyhash
     */
    function oracleToKeyHash(address oracle) external view returns (bytes32) {
        return sOracleToKeyHash[oracle];
    }

    /**
     * @notice Find key oracle associated with given key hash.
     * @return oracle address
     */
    function keyHashToOracle(bytes32 keyHash) external view returns (address) {
        return sKeyHashToOracle[keyHash];
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

    function isValidRequestId(uint256 requestId) internal view returns (bool) {
        if (sRequestIdToCommitment[requestId] != 0) {
            return true;
        } else {
            return false;
        }
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) public view returns (bool) {
        uint256 keyHashesLength = sKeyHashes.length;
        for (uint256 i = 0; i < keyHashesLength; ++i) {
            (uint256 requestId, ) = computeRequestId(sKeyHashes[i], consumer, accId, nonce);
            if (isValidRequestId(requestId)) {
                return true;
            }
        }
        return false;
    }

    /**
     * @notice Returns the proving key hash key associated with this public key
     * @param publicKey the key to return the hash of
     */
    function hashOfKey(uint256[2] memory publicKey) public pure returns (bytes32) {
        return keccak256(abi.encode(publicKey));
    }

    function requestRandomWordsInternal(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords,
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

        if (numWords > MAX_NUM_WORDS) {
            revert NumWordsTooBig(numWords, MAX_NUM_WORDS);
        }

        uint64 nonce = sPrepayment.increaseNonce(accId, msg.sender);
        (uint256 requestId, uint256 preSeed) = computeRequestId(keyHash, msg.sender, accId, nonce);

        sRequestIdToCommitment[requestId] = keccak256(
            abi.encode(requestId, block.number, accId, callbackGasLimit, numWords, msg.sender)
        );
        sRequestOwner[requestId] = msg.sender;

        emit RandomWordsRequested(
            keyHash,
            requestId,
            preSeed,
            accId,
            callbackGasLimit,
            numWords,
            msg.sender,
            isDirectPayment
        );

        return requestId;
    }

    function estimateDirectPaymentFee() internal view returns (uint256) {
        return sDirectPaymentConfig.fulfillmentFee + sDirectPaymentConfig.baseFee;
    }

    function calculateFee(uint64 accId) internal view returns (uint256) {
        uint64 reqCount = sPrepayment.getReqCount(accId);
        uint32 fulfillmentFlatFeeKlayPPM = getFeeTier(reqCount);
        return 1e12 * uint256(fulfillmentFlatFeeKlayPPM);
    }

    function calculateGasCost(uint256 startGas) internal view returns (uint256) {
        return tx.gasprice * (sConfig.gasAfterPaymentCalculation + startGas - gasleft());
    }

    function computeRequestId(
        bytes32 keyHash,
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256, uint256) {
        uint256 preSeed = uint256(keccak256(abi.encode(keyHash, sender, accId, nonce)));
        return (uint256(keccak256(abi.encode(keyHash, preSeed))), preSeed);
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

    function getRandomnessFromProof(
        VRF.Proof memory proof,
        RequestCommitment memory rc
    ) private view returns (bytes32 keyHash, uint256 requestId, uint256 randomness) {
        keyHash = hashOfKey(proof.pk);
        // Only registered proving keys are permitted.
        address oracle = sKeyHashToOracle[keyHash];
        if (oracle == address(0)) {
            revert NoSuchProvingKey(keyHash);
        }
        requestId = uint256(keccak256(abi.encode(keyHash, proof.seed)));
        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }
        if (
            commitment !=
            keccak256(
                abi.encode(
                    requestId,
                    rc.blockNum,
                    rc.accId,
                    rc.callbackGasLimit,
                    rc.numWords,
                    rc.sender
                )
            )
        ) {
            revert IncorrectCommitment();
        }

        bytes32 blockHash = blockhash(rc.blockNum);

        // The seed actually used by the VRF machinery, mixing in the blockhash
        bytes memory actualSeed = abi.encodePacked(
            keccak256(abi.encodePacked(proof.seed, blockHash))
        );
        randomness = VRF.randomValueFromVRFProof(proof, actualSeed); // Reverts on failure
    }
}
