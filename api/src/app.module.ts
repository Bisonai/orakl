import { Module } from '@nestjs/common'
import { FeedController } from './feed/feed.controller'
import { AppService } from './app.service'
import { AppController } from './app.controller'
import { FeedService } from './feed/feed.service'
import { FeedModule } from './feed/feed.module'

@Module({
  imports: [FeedModule],
  controllers: [AppController, FeedController],
  providers: [AppService, FeedService]
})
export class AppModule {}
