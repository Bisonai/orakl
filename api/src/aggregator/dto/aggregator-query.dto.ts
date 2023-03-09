import { ApiProperty } from '@nestjs/swagger'

import { Transform } from 'class-transformer'
import { IsBoolean } from 'class-validator'
import { toBoolean } from '../../common/helper/cast.helper'

export class AggregatorQueryDto {
  @Transform(({ value }) => toBoolean(value))
  @IsBoolean()
  // @ApiProperty({ type: Boolean })
  active?: boolean = false

  @ApiProperty()
  chain?: string
}
