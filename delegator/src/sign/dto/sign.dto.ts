import { ApiProperty } from '@nestjs/swagger'

export class SignDto {
  @ApiProperty()
  from: string

  @ApiProperty()
  to: string

  @ApiProperty()
  input: string

  @ApiProperty()
  gas: string

  @ApiProperty()
  value: string

  @ApiProperty()
  chainId: string

  @ApiProperty()
  gasPrice: string

  @ApiProperty()
  nonce: string

  @ApiProperty()
  v: string

  @ApiProperty()
  r: string

  @ApiProperty()
  s: string

  @ApiProperty()
  rawTx: string

  @ApiProperty()
  signedRawTx?: string
}
