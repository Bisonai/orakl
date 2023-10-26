import { ApiProperty } from '@nestjs/swagger'

export class LastestSubmissionDto {
  @ApiProperty()
  aggregatorHash: string
}
