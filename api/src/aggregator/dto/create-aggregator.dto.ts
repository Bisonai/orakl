import { ApiProperty } from '@nestjs/swagger'

export class CreateAggregatorDto {
  @ApiProperty()
  aggregatorId: string

  @ApiProperty()
  active: boolean

  @ApiProperty()
  name: string

  @ApiProperty()
  heartbeat: number

  @ApiProperty()
  threshold: number

  @ApiProperty()
  absoluteThreshold: number

  @ApiProperty()
  adapterId: number

  @ApiProperty()
  chainId: number
}
