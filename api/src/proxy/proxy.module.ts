import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ProxyController } from './proxy.controller'
import { ProxyService } from './proxy.service'

@Module({
  controllers: [ProxyController],
  providers: [ProxyService, PrismaService]
})
export class ProxyModule {}
