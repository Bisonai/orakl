import { Module } from '@nestjs/common'
import { ErrorService } from './error.service'
import { ErrorController } from './error.controller'
import { PrismaService } from 'src/prisma.service'

@Module({
  controllers: [ErrorController],
  providers: [ErrorService, PrismaService]
})
export class ErrorModule {}
