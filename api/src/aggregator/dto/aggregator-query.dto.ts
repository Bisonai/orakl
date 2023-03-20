import { ApiProperty } from '@nestjs/swagger'
import { Transform } from 'class-transformer'
import { toBoolean } from '../../common/helper/cast.helper'

export class AggregatorQueryDto {
  @Transform(({ value }) => toBoolean(value))
  @ApiProperty()
  chain?: string

  @ApiProperty()
  address: string
}
