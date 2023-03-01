import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  root() {
    return 'Orakl Network API'
  }

  health() {
    return 'OK'
  }
}
