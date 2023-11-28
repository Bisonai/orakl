import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ErrorController } from './error.controller'
import { ErrorService } from './error.service'

@Module({
  controllers: [ErrorController],
  providers: [ErrorService, PrismaService]
})
export class ErrorModule {}
