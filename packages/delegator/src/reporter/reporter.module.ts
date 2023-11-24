import { Module } from '@nestjs/common'
import { ReporterService } from './reporter.service'
import { ReporterController } from './reporter.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [ReporterController],
  providers: [ReporterService, PrismaService]
})
export class ReporterModule {}
