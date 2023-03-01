import { Module } from '@nestjs/common'
import { FeedService } from './feed.service'
import { FeedController } from './feed.controller'

@Module({
  controllers: [FeedController],
  providers: [FeedService]
})
export class FeedModule {}
