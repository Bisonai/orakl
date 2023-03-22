import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  health() {
    return 'Orakl Network Fetcher'
  }
}
