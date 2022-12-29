const { expect } = require("chai");
const { loadFixture } = require("@nomicfoundation/hardhat-network-helpers");

const abi = [
  {
    "inputs": [],
    "stateMutability": "nonpayable",
    "type": "constructor"
  },
  {
    "inputs": [],
    "name": "InsufficientBalance",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      }
    ],
    "name": "InvalidConsumer",
    "type": "error"
  },
  {
    "inputs": [],
    "name": "InvalidSubscription",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "proposedOwner",
        "type": "address"
      }
    ],
    "name": "MustBeRequestedOwner",
    "type": "error"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "owner",
        "type": "address"
      }
    ],
    "name": "MustBeSubOwner",
    "type": "error"
  },
  {
    "inputs": [],
    "name": "PendingRequestExists",
    "type": "error"
  },
  {
    "inputs": [],
    "name": "Reentrant",
    "type": "error"
  },
  {
    "inputs": [],
    "name": "TooManyConsumers",
    "type": "error"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "previousOwner",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "OwnershipTransferred",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "previousAdminRole",
        "type": "bytes32"
      },
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "newAdminRole",
        "type": "bytes32"
      }
    ],
    "name": "RoleAdminChanged",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "account",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "sender",
        "type": "address"
      }
    ],
    "name": "RoleGranted",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "account",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "sender",
        "type": "address"
      }
    ],
    "name": "RoleRevoked",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "to",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "amount",
        "type": "uint256"
      }
    ],
    "name": "SubscriptionCanceled",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      }
    ],
    "name": "SubscriptionConsumerAdded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      }
    ],
    "name": "SubscriptionConsumerRemoved",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "owner",
        "type": "address"
      }
    ],
    "name": "SubscriptionCreated",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "oldBalance",
        "type": "uint256"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "newBalance",
        "type": "uint256"
      }
    ],
    "name": "SubscriptionDecreased",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "oldBalance",
        "type": "uint256"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "newBalance",
        "type": "uint256"
      }
    ],
    "name": "SubscriptionFunded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "from",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "to",
        "type": "address"
      }
    ],
    "name": "SubscriptionOwnerTransferRequested",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "from",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "to",
        "type": "address"
      }
    ],
    "name": "SubscriptionOwnerTransferred",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "DEFAULT_ADMIN_ROLE",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "MAX_CONSUMERS",
    "outputs": [
      {
        "internalType": "uint16",
        "name": "",
        "type": "uint16"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "ORACLE_ROLE",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "WITHDRAWER_ROLE",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "acceptSubscriptionOwnerTransfer",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      }
    ],
    "name": "addConsumer",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "to",
        "type": "address"
      }
    ],
    "name": "cancelSubscription",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "createSubscription",
    "outputs": [
      {
        "internalType": "uint64",
        "name": "",
        "type": "uint64"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "uint96",
        "name": "amount",
        "type": "uint96"
      }
    ],
    "name": "decreaseSubBalance",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "deposit",
    "outputs": [],
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      },
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "getNonce",
    "outputs": [
      {
        "internalType": "uint64",
        "name": "",
        "type": "uint64"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      }
    ],
    "name": "getRoleAdmin",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "internalType": "uint256",
        "name": "index",
        "type": "uint256"
      }
    ],
    "name": "getRoleMember",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      }
    ],
    "name": "getRoleMemberCount",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "getSubOwner",
    "outputs": [
      {
        "internalType": "address",
        "name": "owner",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "getSubscription",
    "outputs": [
      {
        "internalType": "uint96",
        "name": "balance",
        "type": "uint96"
      },
      {
        "internalType": "uint64",
        "name": "reqCount",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "owner",
        "type": "address"
      },
      {
        "internalType": "address[]",
        "name": "consumers",
        "type": "address[]"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "getTotalBalance",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "account",
        "type": "address"
      }
    ],
    "name": "grantRole",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "account",
        "type": "address"
      }
    ],
    "name": "hasRole",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      },
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      }
    ],
    "name": "increaseNonce",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "owner",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "",
        "type": "uint64"
      }
    ],
    "name": "pendingRequestExists",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "consumer",
        "type": "address"
      }
    ],
    "name": "removeConsumer",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "renounceOwnership",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "account",
        "type": "address"
      }
    ],
    "name": "renounceRole",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "requestSubscriptionOwnerTransfer",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "role",
        "type": "bytes32"
      },
      {
        "internalType": "address",
        "name": "account",
        "type": "address"
      }
    ],
    "name": "revokeRole",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes4",
        "name": "interfaceId",
        "type": "bytes4"
      }
    ],
    "name": "supportsInterface",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "transferOwnership",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "typeAndVersion",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint64",
        "name": "subId",
        "type": "uint64"
      },
      {
        "internalType": "uint96",
        "name": "amount",
        "type": "uint96"
      }
    ],
    "name": "withdraw",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "stateMutability": "payable",
    "type": "receive"
  }
]
function parseEther(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18);
}
describe("Prepayment contract", function () {
  async function deployFixture() {
    const PrePayment = await ethers.getContractFactory("Prepayment");
    const [owner, addr1, addr2] = await ethers.getSigners();
    const prePayment = await PrePayment.deploy();
    await prePayment.deployed();

    // Fixtures can return anything you consider useful for your tests
    return { prePayment, owner, addr1, addr2 };
  }
  async function deployMockFixture() {
    [owner, account0, account1, account2] = await ethers.getSigners()


    const Prepayment = await ethers.getContractFactory('Prepayment');
    prepayment = await Prepayment.deploy();
    console.log('prepayment address:', prepayment.address);

    const Coordinator = await ethers.getContractFactory('VRFCoordinator1');
    coordinator = await Coordinator.deploy(prepayment.address);
    console.log('Coordinator address:', coordinator.address);

    let Consumer = await ethers.getContractFactory('VRFConsumerMock')
    consumer = await Consumer.deploy(coordinator.address)
    console.log('VRFConsumerMock Address:', consumer.address)

    await prepayment.createSubscription();

    // Fixtures can return anything you consider useful for your tests
    return { prepayment, owner, coordinator, consumer };
  }

  it.skip("Should create subscription", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("SubscriptionCreated(uint64,address)")
    const transaction = await prePayment.createSubscription();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let SubID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event SubscriptionCreated
        SubID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    expect(SubID).to.be.equal(1)
    console.log(SubID);
    const transactionTemp = await prePayment.getSubscription(1);
    console.log(transactionTemp);
  });
  it.skip("Should add consumer", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("SubscriptionCreated(uint64,address)")
    const transaction = await prePayment.createSubscription();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let SubID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event SubscriptionCreated
        SubID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    //expect(SubID==1)
    console.log(SubID);
    const ownerOfSubID = await prePayment.getSubOwner(SubID)
    console.log(ownerOfSubID)
    //const signer=await  prePayment.connect(owner)
    await prePayment.connect(owner).addConsumer(SubID, "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9")
    await prePayment.connect(owner).addConsumer(SubID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    // console.log(transactionTemp);
    const transactionTemp2 = await prePayment.getSubscription(SubID)
    console.log(transactionTemp2);
    expect(transactionTemp2.consumers.length).to.equal(2)
  });
  it.skip("Should remove consumer", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("SubscriptionCreated(uint64,address)")
    const transaction = await prePayment.createSubscription();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let SubID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event SubscriptionCreated
        SubID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    //expect(SubID==1)
    console.log(SubID);
    await prePayment.connect(owner).addConsumer(SubID, "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9")
    await prePayment.connect(owner).addConsumer(SubID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    let transactionGetInforSub = await prePayment.getSubscription(SubID)
    const lengthComsumerBefore = transactionGetInforSub.consumers.length

    await prePayment.connect(owner).removeConsumer(SubID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    // console.log(transactionTemp);
    transactionGetInforSub = await prePayment.getSubscription(SubID)
    const lengthComsumerAfter = transactionGetInforSub.consumers.length
    //console.log(transactionTemp2);
    expect(lengthComsumerBefore).to.be.greaterThan(lengthComsumerAfter)
  });
  it.skip("Should deposit", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("SubscriptionCreated(uint64,address)")
    const transaction = await prePayment.createSubscription();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let SubID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event SubscriptionCreated
        SubID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    const balanceBefore = await prePayment.getSubscription(SubID);
    //expect(SubID==1)
    const transactionDeposit = await prePayment.deposit(SubID, { value: 1000000000000000 });
    const balanceAfter = await prePayment.getSubscription(SubID);
    expect(balanceAfter.balance).to.be.greaterThan(balanceBefore.balance)
  });
  it.skip("Should withdraw", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("SubscriptionCreated(uint64,address)")
    const transaction = await prePayment.createSubscription();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let SubID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event SubscriptionCreated
        SubID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
   // const expected2 = ethers.BigNumber.from((await ethers.provider.getBalance(owner.address)))
    //Deposit
    const transactionDeposit = await prePayment.deposit(SubID, { value: 100000 });
    //Check balance Before & After
    const balanceOwnerBefore = parseInt((ethers.BigNumber.from((await ethers.provider.getBalance(owner.address)))).toString());
    const balanceSubBefore = (await prePayment.getSubscription(SubID)).balance;
    //Withdraw
    const txWithdraw = await prePayment.connect(owner).withdraw(SubID, 50000);
    const txRecip = await txWithdraw.wait();

    const balanceOwnerAfter = parseInt((ethers.BigNumber.from((await ethers.provider.getBalance(owner.address)))).toString())
    const balanceSubAfter = (await prePayment.getSubscription(SubID)).balance;

    //const string = parseInt( balanceOwnerAfter.toString());

    //console.log(string); 


    // expect((balanceOwnerAfter)).to.be.greaterThan((balanceOwnerBefore))
    //0x021e19d0bc154a512231
    //const expected = ethers.BigNumber.from(balanceOwnerAfter.toString());//0x021e19d01acbf980115e
    //const actual = ethers.BigNumber.from(balanceOwnerBefore.toString());//0x021e19d052fc346f5499
    //expect(balanceOwnerAfter.toNumber().gt(balanceOwnerBefore.toNumber())).to.be.true;
    expect(balanceOwnerAfter).to.be.greaterThan(balanceOwnerBefore)
    //expect(expected.isEqual(actual)).to.be.true;
  });
  it.skip('Should cancel subscription, pending tx', async function () {
    const { prepayment, owner } = await loadFixture(
      deployMockFixture
    );
    await prepayment.cancelSubscription(1, owner.address);
  })
  it.skip('Should not cancel subscription with pending tx', async function () {
    const { prepayment, owner, coordinator, consumer } = await loadFixture(
      deployMockFixture
    );
    // Register Proving Key
    const oracle = owner.address // Hardhat account 19
    const publicProvingKey = [
      '95162740466861161360090244754314042169116280320223422208903791243647772670481',
      '53113177277038648369733569993581365384831203706597936686768754351087979105423'
    ]
    await coordinator.registerProvingKey(oracle, publicProvingKey);
    const minimumRequestConfirmations = 3
    const maxGasLimit = 1_000_000
    const gasAfterPaymentCalculation = 1_000
    const feeConfig = {
      fulfillmentFlatFeeLinkPPMTier1: 0,
      fulfillmentFlatFeeLinkPPMTier2: 0,
      fulfillmentFlatFeeLinkPPMTier3: 0,
      fulfillmentFlatFeeLinkPPMTier4: 0,
      fulfillmentFlatFeeLinkPPMTier5: 0,
      reqsForTier2: 0,
      reqsForTier3: 0,
      reqsForTier4: 0,
      reqsForTier5: 0
    }

    // Configure VRF Coordinator
    await coordinator.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    )

    const subId = 1
    await prepayment.addConsumer(subId, consumer.address);
    await prepayment.addCoordinator(coordinator.address);

    await consumer.requestRandomWords();

    await expect(prepayment.cancelSubscription(1, owner.address)).to.be.revertedWithCustomError(prepayment, 'PendingRequestExists');
  })
  it('Should remove Coordinator', async function () {
    const { prepayment, owner, coordinator, consumer } = await loadFixture(
      deployMockFixture
    );
    // Register Proving Key
    const oracle = owner.address // Hardhat account 19
    const publicProvingKey = [
      '95162740466861161360090244754314042169116280320223422208903791243647772670481',
      '53113177277038648369733569993581365384831203706597936686768754351087979105423'
    ]
    await coordinator.registerProvingKey(oracle, publicProvingKey);
    const minimumRequestConfirmations = 3
    const maxGasLimit = 1_000_000
    const gasAfterPaymentCalculation = 1_000
    const feeConfig = {
      fulfillmentFlatFeeLinkPPMTier1: 0,
      fulfillmentFlatFeeLinkPPMTier2: 0,
      fulfillmentFlatFeeLinkPPMTier3: 0,
      fulfillmentFlatFeeLinkPPMTier4: 0,
      fulfillmentFlatFeeLinkPPMTier5: 0,
      reqsForTier2: 0,
      reqsForTier3: 0,
      reqsForTier4: 0,
      reqsForTier5: 0
    }

    // Configure VRF Coordinator
    await coordinator.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    )

    const subId = 1
    await prepayment.addConsumer(subId, consumer.address);
    await prepayment.addCoordinator(coordinator.address);
    const tx=await (await prepayment.removeCoordinator(coordinator.address)).wait()
    //const balanceSubBefore = await prepayment.getSubscription(subId);
    expect(tx.status).to.equal(1)
  })
});