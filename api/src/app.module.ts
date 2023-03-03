import { Module } from '@nestjs/common'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { FeedModule } from './feed/feed.module'
import { FeedController } from './feed/feed.controller'
import { FeedService } from './feed/feed.service'
import { ChainModule } from './chain/chain.module'
import { ChainController } from './chain/chain.controller'
import { ChainService } from './chain/chain.service'
import { AdapterModule } from './adapter/adapter.module'
import { AdapterController } from './adapter/adapter.controller'
import { AdapterService } from './adapter/adapter.service'
import { PrismaService } from './prisma.service'

@Module({
  imports: [ChainModule, AdapterModule, FeedModule],
  controllers: [AppController, AdapterController, FeedController],
  providers: [AppService, PrismaService, AdapterService, FeedService]
})
export class AppModule {}
