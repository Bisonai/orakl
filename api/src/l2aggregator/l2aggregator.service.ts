import { Injectable } from '@nestjs/common'
import { PrismaService } from 'src/prisma.service'

@Injectable()
export class L2aggregatorService {
  constructor(private readonly prismaService: PrismaService) {}
  async l2Address(l1Address: string, chainId: bigint) {
    return await this.prismaService.l2AggregatorPair.findFirst({
      where: { l1AggregatorAddress: l1Address, chainId }
    })
  }
}
