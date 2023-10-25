import { ApiProperty } from '@nestjs/swagger'

export class LastSubmissionDto {
  @ApiProperty()
  aggregatorId: bigint

  @ApiProperty()
  value: bigint
}
