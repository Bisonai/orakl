import { Injectable, HttpStatus, HttpException, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { CreateVrfKeyDto } from './dto/create-vrf-key.dto'
import { UpdateVrfKeyDto } from './dto/update-vrf-key.dto'

@Injectable()
export class VrfService {
  private readonly logger = new Logger(VrfService.name)

  constructor(private prisma: PrismaService) {}

  async create(createVrfKeyDto: CreateVrfKeyDto) {
    // chain
    const chainName = createVrfKeyDto.chain
    const chain = await this.prisma.chain.findUnique({
      where: { name: chainName }
    })

    if (chain == null) {
      const msg = `chain.name [${chainName}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    const data: Prisma.VrfKeyUncheckedCreateInput = {
      sk: createVrfKeyDto.sk,
      pk: createVrfKeyDto.pk,
      pkX: createVrfKeyDto.pkX,
      pkY: createVrfKeyDto.pkY,
      keyHash: createVrfKeyDto.keyHash,
      chainId: chain.id
    }

    return await this.prisma.vrfKey.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.VrfKeyWhereUniqueInput
    where?: Prisma.VrfKeyWhereInput
    orderBy?: Prisma.VrfKeyOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.vrfKey.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(VrfKeyWhereUniqueInput: Prisma.VrfKeyWhereUniqueInput) {
    return await this.prisma.vrfKey.findUnique({
      where: VrfKeyWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.VrfKeyWhereUniqueInput; updateVrfKeyDto: UpdateVrfKeyDto }) {
    const { where, updateVrfKeyDto } = params
    return await this.prisma.vrfKey.update({
      data: updateVrfKeyDto,
      where
    })
  }

  async remove(where: Prisma.VrfKeyWhereUniqueInput) {
    return await this.prisma.vrfKey.delete({
      where
    })
  }
}
