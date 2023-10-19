import { ApiProperty } from '@nestjs/swagger'

export class LatestSubmittionDto {
  @ApiProperty()
  aggregatorHash: string
}
