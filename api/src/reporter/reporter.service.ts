import { Injectable, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { getChain, getService } from '../common/utils'
import { PrismaService } from '../prisma.service'
import { CreateReporterDto } from './dto/create-reporter.dto'
import { UpdateReporterDto } from './dto/update-reporter.dto'
import { decryptText, encryptText, flattenReporter } from './reporter.utils'

@Injectable()
export class ReporterService {
  private readonly logger = new Logger(ReporterService.name)

  constructor(private prisma: PrismaService) {}

  async create(createReporterDto: CreateReporterDto) {
    // chain
    const chainName = createReporterDto.chain
    const chain = await getChain({ chain: this.prisma.chain, chainName, logger: this.logger })

    // service
    const serviceName = createReporterDto.service
    const service = await getService({
      service: this.prisma.service,
      serviceName,
      logger: this.logger
    })

    // reporter
    const data: Prisma.ReporterUncheckedCreateInput = {
      address: createReporterDto.address,
      privateKey: await encryptText(createReporterDto.privateKey),
      oracleAddress: createReporterDto.oracleAddress,
      chainId: chain.id,
      serviceId: service.id
    }

    return await this.prisma.reporter.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ReporterWhereUniqueInput
    where?: Prisma.ReporterWhereInput
    orderBy?: Prisma.ReporterOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    const reporters = await this.prisma.reporter.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy,
      select: {
        id: true,
        address: true,
        privateKey: true,
        oracleAddress: true,
        service: { select: { name: true } },
        chain: { select: { name: true } }
      }
    })

    return await Promise.all(
      reporters.map(async (L) => {
        return await flattenReporter(L)
      })
    )
  }

  async findOne(reporterWhereUniqueInput: Prisma.ReporterWhereUniqueInput) {
    const reporter = await this.prisma.reporter.findUnique({
      where: reporterWhereUniqueInput,
      select: {
        id: true,
        address: true,
        privateKey: true,
        oracleAddress: true,
        service: { select: { name: true } },
        chain: { select: { name: true } }
      }
    })

    return await flattenReporter(reporter)
  }

  async update(params: {
    where: Prisma.ReporterWhereUniqueInput
    updateReporterDto: UpdateReporterDto
  }) {
    const { where, updateReporterDto } = params
    return await this.prisma.reporter.update({
      data: updateReporterDto,
      where
    })
  }

  async remove(where: Prisma.ReporterWhereUniqueInput) {
    return await this.prisma.reporter.delete({
      where
    })
  }
  async encrypt(text: string) {
    return await encryptText(text)
  }
  async decrypt(text: string) {
    return await decryptText(text)
  }
}
