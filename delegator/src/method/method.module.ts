import { Module } from '@nestjs/common'
import { MethodService } from './method.service'
import { MethodController } from './method.controller'
import { PrismaService } from 'src/prisma.service'

@Module({
  controllers: [MethodController],
  providers: [MethodService, PrismaService]
})
export class MethodModule {}
