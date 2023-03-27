import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { OrganizationDto } from './dto/organization.dto'

@Injectable()
export class OrganizationService {
  constructor(private prisma: PrismaService) {}

  async create(organizationDto: OrganizationDto) {
    return await this.prisma.organization.create({ data: organizationDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.OrganizationWhereUniqueInput
    where?: Prisma.OrganizationWhereInput
    orderBy?: Prisma.OrganizationOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.organization.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(organizationWhereUniqueInput: Prisma.OrganizationWhereUniqueInput) {
    return await this.prisma.organization.findUnique({
      where: organizationWhereUniqueInput
    })
  }

  async update(params: {
    where: Prisma.OrganizationWhereUniqueInput
    organizationDto: OrganizationDto
  }) {
    const { where, organizationDto } = params
    return await this.prisma.organization.update({
      data: organizationDto,
      where
    })
  }

  async remove(where: Prisma.OrganizationWhereUniqueInput) {
    return await this.prisma.organization.delete({
      where
    })
  }
}
