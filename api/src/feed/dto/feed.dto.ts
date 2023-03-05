import { ApiProperty } from '@nestjs/swagger'
import { Prisma } from 'prisma'

export class FeedDto {
  @ApiProperty()
  name: string

  @ApiProperty()
  definition: Prisma.JsonObject
}
