import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class RedisConfigService {
  constructor(private configService: ConfigService) {}

    get vrf() {
    console.log('redis host calling')
    return this.configService.get('redis.vrf');
    }
  
    get requestResponse() {
      console.log('redis host calling')
      return this.configService.get('redis.reqeustResponse')
    }  

    get aggregator() {
      return this.configService.get('redis.aggregator')
    }    

    get fetcher() {
      console.log('redis host calling')
      return this.configService.get('redis.fetcher')
    }      
  
}
