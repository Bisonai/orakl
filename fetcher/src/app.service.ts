import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  root() {
    return 'Orakl Network Fetcher'
  }

  health() {
    return 'OK'
  }
}
