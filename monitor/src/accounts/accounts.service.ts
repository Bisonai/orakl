import { Injectable } from "@nestjs/common";
import { PoolClient } from "pg";
import { AccountBalanceRepository } from "./accounts.repository";
import { Account } from "./entities/account.entity";
import { ethers } from "ethers";
import { CommonConfigService } from "src/common/common.config";
import { BalanceDTO } from "./dto/balance.dto";
import { MonitorConfigService } from "src/monitor.config/monitor.config.service";
import { MONITOR_CONFIG } from "src/common/types";
import { IncomingWebhook } from "@slack/webhook";
import { P } from "pino";

@Injectable()
export class AccountsService {
  private monitorClient: PoolClient;

  constructor(
    private readonly accountBalanceRepository: AccountBalanceRepository,
    private readonly commonConfigService: CommonConfigService,
    private readonly monitorConfigService: MonitorConfigService
  ) {}

  async getAccountList(): Promise<[Account] | null> {
    return await this.accountBalanceRepository.getAllAccount();
  }

  async setBalance(accountInfo, balance): Promise<void> {
    try {
      const data: BalanceDTO = {
        address: accountInfo.address,
        balance: balance,
      };
      const result = await this.accountBalanceRepository.upsertBalance(data);
    } catch (e) {
      console.log(e);
    }
  }

  async cronUpdateBalance() {
    const accountLists: [Account] = await this.getAccountList();
    if (accountLists.length > 0) {
      for (const accountInfo of accountLists) {
        try {
          const Provider = new ethers.providers.JsonRpcProvider(
            this.commonConfigService.provider
          );
          const balanceBigInt = await Provider.getBalance(accountInfo.address);
          if (balanceBigInt) {
            const balance = ethers.utils.formatEther(balanceBigInt);
            await this.setBalance(accountInfo, balance);
          }
        } catch (e) {
          console.log(e);
        }
      }
    }
  }

  async createAccount(data: Account) {
    await this.accountBalanceRepository.insertAccount(data);
  }

  async cronAlarmLowBalance() {
    const accountLists: [Account] = await this.getAccountList();
    
    if (accountLists.length > 0) {
      for (const accountInfo of accountLists) {
        try {
          const balance = await this.accountBalanceRepository.getBalance(accountInfo.address);
          const balance_alarm_amount = await this.accountBalanceRepository.getBalanceAlarmAmount(accountInfo.address);
          
          if (balance && balance_alarm_amount) {
            const isEnabled = (balance_alarm_amount !== 0);

            if (isEnabled && balance < balance_alarm_amount) {
              this.sendToSlackLowBalance(accountInfo, balance, balance_alarm_amount);
            }
          }
        } catch (e) {
          console.error(e);
        }
      }
    }
  }

  async sendToSlackLowBalance(account: Account, balance, minBalance) {
    const { value } = await this.monitorConfigService.getValueByName('slack_url');
    const webhook = new IncomingWebhook(value)

    const date = new Date();
    const month = date.toLocaleString('en-US', { month: 'long' });    
    const day = date.getDate();
    const year = date.getFullYear();
    const hour = date.getHours();
    const minute = date.getMinutes();

    
    const deployedEnv = process.env.NODE_ENV;
    const isValidDeployedEnv = !(['CYPRESS', 'BAOBAB', 'DEV'].includes(deployedEnv));
    const prefix = isValidDeployedEnv ? `[${deployedEnv}]` : '';
    if (isValidDeployedEnv) {
      console.error(`
      Invalid deployed environment. The NODE_ENV must be set like CYPRESS, BAOBAB or DEV.
      Currently, the NODE_ENV value is ${deployedEnv}
      `);
    }

    const headerText = `${prefix}  :coin:  Low Account Balance in ${account.name}`;
    const dateText = `${month} ${day}, ${year} ${hour}:${minute}   |   Balance Report`;
    const queueNameText = `:herb: account: *${account.address}*`;
    const context = `\`${account.name}\` has *${balance}* Klay. \nMinimum balance is ${minBalance} Klay.`;


    if (isValidDeployedEnv) {
      await webhook.send(
        {
          blocks: [
          {
            "type": "header",
            "text": {
              "type": "plain_text",
              "text": headerText,
            }
          },
          {
            "type": "context",
            "elements": [
              {
                "text": dateText,
                "type": "mrkdwn"
              }
            ]
          },        
          {
            "type": "divider"
          },        
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": queueNameText,
            }
          },        
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": context,
            }
          },
          {
            "type": "divider"
            },    
            {
              "type": "context",
              "elements": [
                {
                  "type": "image",
                  "image_url": "https://www.orakl.network/favicon.ico",
                  "alt_text": "orakl network"
                },
                {
                  "type": "mrkdwn",
                  "text": " Developed by Bisonai Infra Team"
                },
              ]
            },     
            {
              "type": "divider"
            },       
        ]
      })
    }
  }    
}
