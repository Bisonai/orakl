import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ListenerController } from './listener.controller'
import { ListenerService } from './listener.service'

@Module({
  controllers: [ListenerController],
  providers: [ListenerService, PrismaService]
})
export class ListenerModule {}
