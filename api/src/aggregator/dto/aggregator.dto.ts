import { ApiProperty } from '@nestjs/swagger'
import { ChainDto as Chain } from '../../chain/dto/chain.dto'
import { AdapterDto as Adapter } from '../../adapter/dto/adapter.dto'

export class AggregatorDto {
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
  adapterId: string

  @ApiProperty()
  chainName: string
}
