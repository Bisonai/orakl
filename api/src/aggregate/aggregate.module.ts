import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { AggregateController } from './aggregate.controller'
import { AggregateService } from './aggregate.service'

@Module({
  controllers: [AggregateController],
  providers: [AggregateService, PrismaService]
})
export class AggregateModule {}
