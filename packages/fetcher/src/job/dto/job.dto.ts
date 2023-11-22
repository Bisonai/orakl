import { ApiProperty } from '@nestjs/swagger'

export class JobDto {
  @ApiProperty()
  aggregatorId: string
}
