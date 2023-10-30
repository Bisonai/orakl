import { Controller, Get, HttpException, HttpStatus, Param } from '@nestjs/common'
import { L2aggregatorService } from './l2aggregator.service'
import { ChainService } from 'src/chain/chain.service'

@Controller({
  path: 'l2aggregator',
  version: '1'
})
export class L2aggregatorController {
  constructor(
    private readonly l2AggregatorService: L2aggregatorService,
    private readonly chainService: ChainService
  ) {}
  @Get(':chain/:l1Address')
  async getL2Address(@Param('chain') chain: string, @Param('l1Address') l1Address: string) {
    // chain
    const chainRecord = await this.chainService.findOne({ name: chain })
    if (chainRecord == null) {
      const msg = `Chain [${chain}] not found`
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }
    return await this.l2AggregatorService.l2Address(l1Address, chainRecord.id)
  }
}
