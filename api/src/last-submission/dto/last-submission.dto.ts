import { ApiProperty } from '@nestjs/swagger'

export class LastSubmissionDto {
  @ApiProperty()
  aggregatorId: number

  @ApiProperty()
  value: number
}
