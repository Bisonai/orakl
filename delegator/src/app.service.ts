import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  root() {
    return 'Orakl Network Delegator API'
  }

  health() {
    return 'OK'
  }
}
