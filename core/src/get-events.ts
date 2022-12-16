import { Queue } from 'bullmq';
import { Contract, ethers } from 'ethers'
import { BULLMQ_CONNECTION } from './settings';
import {
  IListenerBlock
} from './types'
import { readTextFile, writeTextFile } from './utils'
export class get_event {
  iface: ethers.utils.Interface;
  queue: Queue<any, any, string>;
  fn: any;
  emit_contract: Contract;
  listener_block: IListenerBlock;
  provider: ethers.providers.JsonRpcProvider;
  eventName: string;
  constructor(
    provider: ethers.providers.JsonRpcProvider,
    listenerAddress: string,
    queueName: string,
    eventName: string,
    abis: Array<object>,
    blockFilePath: string,
    wrapFn) {
    this.provider = provider;
    this.iface = new ethers.utils.Interface(abis);
    this.queue = new Queue(queueName, BULLMQ_CONNECTION)
    this.fn = wrapFn(this.iface, this.queue)

    console.debug(`listenToEvents:topicId ${eventName}`)
    this.eventName = eventName;

    this.emit_contract = new ethers.Contract(listenerAddress, abis, provider);
    this.listener_block = {
      startBlock: 0,
      filePath: blockFilePath
    }
  }

  async get_events() {
    try {
      if (this.listener_block.startBlock <= 0) {
        let start = "0";
        try {
          start = await readTextFile(this.listener_block.filePath);
        } catch {
        }
        this.listener_block.startBlock = parseInt(start);
      }
      const latest_block = await this.provider.getBlockNumber();

      if (latest_block >= this.listener_block.startBlock) {

        const events = await this.emit_contract.queryFilter(this.eventName, this.listener_block.startBlock, latest_block);
        //save last block here
        console.log(this.listener_block.startBlock, ' - ', latest_block);
        this.listener_block.startBlock = latest_block + 1;
        await writeTextFile(this.listener_block.filePath, this.listener_block.startBlock.toString());
        if (events?.length > 0) {
          events.forEach(this.fn);
        }
      }
    } catch (error) {
      console.log('vrt listener', error);
    }
  }
}

export async function get_events(eventName: string, emit_contract: Contract, provider: ethers.providers.JsonRpcProvider, listener_block: IListenerBlock) {
  try {
    if (listener_block.startBlock <= 0) {
      let start = "0";
      try {
        start = await readTextFile(listener_block.filePath);
      } catch {
      }
      listener_block.startBlock = parseInt(start);
    }
    const latest_block = await provider.getBlockNumber();
    if (latest_block >= listener_block.startBlock) {
      const events = await emit_contract.queryFilter(eventName, listener_block.startBlock, latest_block);
      //save last block here
      console.log(listener_block.startBlock, ' - ', latest_block);
      listener_block.startBlock = latest_block + 1;
      await writeTextFile(listener_block.filePath, listener_block.startBlock.toString());
      return events;
    }
  } catch (error) {
  }
  return [];
}


