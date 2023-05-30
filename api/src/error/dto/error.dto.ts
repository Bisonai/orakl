import { ApiProperty } from '@nestjs/swagger'

export class ErrorDto {
  @ApiProperty()
  requestId: string

  @ApiProperty()
  timestamp: string

  @ApiProperty()
  errorMsg: string
}
