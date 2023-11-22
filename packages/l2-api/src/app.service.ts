import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  health(): string {
    return 'Orakl L2 Config Api!'
  }
}
