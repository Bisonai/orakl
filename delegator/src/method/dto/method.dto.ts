import { ApiProperty } from '@nestjs/swagger'

export class MethodDto {
  @ApiProperty()
  name: string

  @ApiProperty()
  contractId: number
}
