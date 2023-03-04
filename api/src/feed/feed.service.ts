import { Injectable } from '@nestjs/common'
import { Feed, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'

@Injectable()
export class FeedService {
  constructor(private prisma: PrismaService) {}

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.FeedWhereUniqueInput
    where?: Prisma.FeedWhereInput
    orderBy?: Prisma.FeedOrderByWithRelationInput
  }): Promise<Feed[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.feed.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(feedWhereUniqueInput: Prisma.FeedWhereUniqueInput): Promise<Feed | null> {
    return this.prisma.feed.findUnique({
      where: feedWhereUniqueInput
    })
  }
}
