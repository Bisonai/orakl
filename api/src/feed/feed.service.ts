import { Injectable } from '@nestjs/common'

@Injectable()
export class FeedService {
  findAll() {
    return `This action returns all feed`
  }

  findOne(id: number) {
    return `This action returns a #${id} feed`
  }
}
