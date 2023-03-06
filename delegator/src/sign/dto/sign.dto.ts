import { ApiProperty } from '@nestjs/swagger'

export class SignDto {
  @ApiProperty()
  txHash: string
}
