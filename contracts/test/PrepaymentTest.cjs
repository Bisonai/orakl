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

    await prepayment.createAccount();

    return { prepayment, owner, coordinator, consumer };
  }

  it("Should create Account", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("AccountCreated(uint64,address)")
    const transaction = await prePayment.createAccount();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let AccID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event AccountCreated
        AccID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    expect(AccID).to.be.equal(1)
    console.log(AccID);
    const transactionTemp = await prePayment.getAccount(1);
    console.log(transactionTemp);
  });
  it("Should add consumer", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("AccountCreated(uint64,address)")
    const transaction = await prePayment.createAccount();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let AccID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event AccountCreated
        AccID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    //expect(AccID==1)
    console.log(AccID);
    const ownerOfAccID = await prePayment.getAccOwner(AccID)
    console.log(ownerOfAccID)
    //const signer=await  prePayment.connect(owner)
    await prePayment.connect(owner).addConsumer(AccID, "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9")
    await prePayment.connect(owner).addConsumer(AccID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    // console.log(transactionTemp);
    const transactionTemp2 = await prePayment.getAccount(AccID)
    console.log(transactionTemp2);
    expect(transactionTemp2.consumers.length).to.equal(2)
  });
  it("Should remove consumer", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("AccountCreated(uint64,address)")
    const transaction = await prePayment.createAccount();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let AccID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event AccountCreated
        AccID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    //expect(AccID==1)
    console.log(AccID);
    await prePayment.connect(owner).addConsumer(AccID, "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9")
    await prePayment.connect(owner).addConsumer(AccID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    let transactionGetInforAcc = await prePayment.getAccount(AccID)
    const lengthComsumerBefore = transactionGetInforAcc.consumers.length

    await prePayment.connect(owner).removeConsumer(AccID, "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
    // console.log(transactionTemp);
    transactionGetInforAcc = await prePayment.getAccount(AccID)
    const lengthComsumerAfter = transactionGetInforAcc.consumers.length
    //console.log(transactionTemp2);
    expect(lengthComsumerBefore).to.be.greaterThan(lengthComsumerAfter)
  });
  it("Should deposit", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("AccountCreated(uint64,address)")
    const transaction = await prePayment.createAccount();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let AccID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event AccountCreated
        AccID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }
    const balanceBefore = await prePayment.getAccount(AccID);
    //expect(AccID==1)
    const transactionDeposit = await prePayment.deposit(AccID, { value: 1000000000000000 });
    const balanceAfter = await prePayment.getAccount(AccID);
    expect(balanceAfter.balance).to.be.greaterThan(balanceBefore.balance)
  });
  it("Should withdraw", async function () {
    const { prePayment, owner, addr1, addr2 } = await loadFixture(
      deployFixture
    );
    const functionSignature = ethers.utils.id("AccountCreated(uint64,address)")
    const transaction = await prePayment.createAccount();
    const transactionRe = await transaction.wait()
    const logs = transactionRe.logs
    let AccID
    for (const log of logs) {
      if (log.topics[0] === functionSignature) {
        //1 is index arguments in event AccountCreated
        AccID = parseInt(log.topics[1], 16)
        console.log(parseInt(log.topics[1], 16))
      }
    }

    //Deposit
    const transactionDeposit = await prePayment.deposit(AccID, { value: 100000 });
    //Check balance Before & After
    const balanceOwnerBefore = parseInt((ethers.BigNumber.from((await ethers.provider.getBalance(owner.address)))).toString());
    const balanceAccBefore = (await prePayment.getAccount(AccID)).balance;
    //Withdraw
    const txWithdraw = await prePayment.connect(owner).withdraw(AccID, 50000);
    const txRecip = await txWithdraw.wait();

    const balanceOwnerAfter = parseInt((ethers.BigNumber.from((await ethers.provider.getBalance(owner.address)))).toString())
    const balanceAccAfter = (await prePayment.getAccount(AccID)).balance;
    expect(balanceOwnerAfter).to.be.greaterThan(balanceOwnerBefore)

  });
  it('Should cancel Account, pending tx', async function () {
    const { prepayment, owner } = await loadFixture(
      deployMockFixture
    );
    await prepayment.cancelAccount(1, owner.address);
  })
  it('Should not cancel Account with pending tx', async function () {
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

    const AccId = 1
    await prepayment.addConsumer(AccId, consumer.address);
    await prepayment.addCoordinator(coordinator.address);

    await consumer.requestRandomWords();

    await expect(prepayment.cancelAccount(1, owner.address)).to.be.revertedWithCustomError(prepayment, 'PendingRequestExists');
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

    const AccId = 1
    await prepayment.addConsumer(AccId, consumer.address);
    await prepayment.addCoordinator(coordinator.address);
    const tx=await (await prepayment.removeCoordinator(coordinator.address)).wait()
    expect(tx.status).to.equal(1)
  })
});