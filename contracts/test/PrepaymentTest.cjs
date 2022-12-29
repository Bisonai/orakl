const { expect } = require("chai");
const { loadFixture } = require("@nomicfoundation/hardhat-network-helpers");

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

    return { prepayment, owner, coordinator, consumer };
  }

  it("Should create subscription", async function () {
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
  it("Should add consumer", async function () {
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
  it("Should remove consumer", async function () {
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
  it("Should deposit", async function () {
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
  it("Should withdraw", async function () {
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
    expect(balanceOwnerAfter).to.be.greaterThan(balanceOwnerBefore)

  });
  it('Should cancel subscription, pending tx', async function () {
    const { prepayment, owner } = await loadFixture(
      deployMockFixture
    );
    await prepayment.cancelSubscription(1, owner.address);
  })
  it('Should not cancel subscription with pending tx', async function () {
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
    expect(tx.status).to.equal(1)
  })
});