import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { VrfController } from './vrf.controller'
import { VrfService } from './vrf.service'

@Module({
  controllers: [VrfController],
  providers: [VrfService, PrismaService]
})
export class VrfModule {}
