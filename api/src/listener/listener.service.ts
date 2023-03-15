import { Injectable, HttpStatus, HttpException, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { CreateListenerDto } from './dto/create-listener.dto'
import { UpdateListenerDto } from './dto/update-listener.dto'

@Injectable()
export class ListenerService {
  private readonly logger = new Logger(ListenerService.name)

  constructor(private prisma: PrismaService) {}

  async create(createListenerDto: CreateListenerDto) {
    // chain
    const chainName = createListenerDto.chain
    const chain = await this.prisma.chain.findUnique({
      where: { name: chainName }
    })

    if (chain == null) {
      const msg = `chain.name [${chainName}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    // chain
    const serviceName = createListenerDto.service
    const service = await this.prisma.service.findUnique({
      where: { name: serviceName }
    })

    if (service == null) {
      const msg = `service.name [${serviceName}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    const data: Prisma.ListenerUncheckedCreateInput = {
      address: createListenerDto.address,
      eventName: createListenerDto.eventName,
      chainId: chain.id,
      serviceId: service.id
    }

    return await this.prisma.listener.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ListenerWhereUniqueInput
    where?: Prisma.ListenerWhereInput
    orderBy?: Prisma.ListenerOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.listener.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(listenerWhereUniqueInput: Prisma.ListenerWhereUniqueInput) {
    return await this.prisma.listener.findUnique({
      where: listenerWhereUniqueInput
    })
  }

  async update(params: {
    where: Prisma.ListenerWhereUniqueInput
    updateListenerDto: UpdateListenerDto
  }) {
    const { where, updateListenerDto } = params
    return await this.prisma.listener.update({
      data: updateListenerDto,
      where
    })
  }

  async remove(where: Prisma.ListenerWhereUniqueInput) {
    return await this.prisma.listener.delete({
      where
    })
  }
}
