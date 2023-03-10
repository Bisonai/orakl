import { ApiProperty } from '@nestjs/swagger'

export class AggregatorDto {
  @ApiProperty()
  aggregatorHash: string

  @ApiProperty()
  active: boolean

  @ApiProperty()
  name: string

  @ApiProperty()
  address: string

  @ApiProperty()
  heartbeat: number

  @ApiProperty()
  threshold: number

  @ApiProperty()
  absoluteThreshold: number

  @ApiProperty()
  adapterHash: string

  @ApiProperty()
  chain: string
}
