import { Logger, Injectable } from '@nestjs/common'
import { CreateFeedDto } from './dto/create-feed.dto'
import { UpdateFeedDto } from './dto/update-feed.dto'

@Injectable()
export class FeedService {
  private readonly logger = new Logger(FeedService.name)

  create(createFeedDto: CreateFeedDto) {
    return 'This action adds a new feed'
  }

  findAll() {
    return `This action returns all feed`
  }

  findOne(id: string) {
    // FIXME
    const value = 123456789
    this.logger.log(`findOne ${value}`)
    return value
  }

  update(id: number, updateFeedDto: UpdateFeedDto) {
    return `This action updates a #${id} feed`
  }

  remove(id: number) {
    return `This action removes a #${id} feed`
  }
}
