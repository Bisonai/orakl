import { ApiProperty } from '@nestjs/swagger'
import { FeedDto as Feed } from '../../feed/dto/feed.dto'

export class CreateAdapterDto {
  @ApiProperty()
  adapterId: string

  @ApiProperty()
  name: string

  @ApiProperty()
  decimals: number

  @ApiProperty({ type: () => [Feed] })
  feeds: Feed[]
}
