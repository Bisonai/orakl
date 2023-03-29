import { Module } from '@nestjs/common'
import { FunctionService } from './function.service'
import { FunctionController } from './function.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [FunctionController],
  providers: [FunctionService, PrismaService]
})
export class FunctionModule {}
