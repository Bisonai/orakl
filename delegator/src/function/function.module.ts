import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { FunctionController } from './function.controller'
import { FunctionService } from './function.service'

@Module({
  controllers: [FunctionController],
  providers: [FunctionService, PrismaService]
})
export class FunctionModule {}
