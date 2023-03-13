import { Module } from '@nestjs/common'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import { SignController } from './sign.controller'

@Module({
  controllers: [SignController],
  providers: [SignService, PrismaService]
})
export class SignModule {}
