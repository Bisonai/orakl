import { Module } from '@nestjs/common'
import { VrfService } from './vrf.service'
import { VrfController } from './vrf.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [VrfController],
  providers: [VrfService, PrismaService]
})
export class VrfModule {}
