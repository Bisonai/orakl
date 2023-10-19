import { ApiProperty } from '@nestjs/swagger'

export class LastSubmissionDto {
  @ApiProperty()
  aggregatorId: bigint

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number
}
