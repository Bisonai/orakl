import { Module } from '@nestjs/common'
import { ProxyService } from './proxy.service'
import { ProxyController } from './proxy.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [ProxyController],
  providers: [ProxyService, PrismaService]
})
export class ProxyModule {}
