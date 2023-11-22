import { Module } from '@nestjs/common'
import { ListenerService } from './listener.service'
import { ListenerController } from './listener.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [ListenerController],
  providers: [ListenerService, PrismaService]
})
export class ListenerModule {}
