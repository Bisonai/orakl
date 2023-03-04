import { Module } from '@nestjs/common'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { FeedModule } from './feed/feed.module'
import { FeedController } from './feed/feed.controller'
import { FeedService } from './feed/feed.service'
import { ChainModule } from './chain/chain.module'
import { AdapterModule } from './adapter/adapter.module'
import { AdapterController } from './adapter/adapter.controller'
import { AdapterService } from './adapter/adapter.service'
import { PrismaService } from './prisma.service'
import { AggregatorModule } from './aggregator/aggregator.module'
import { AggregatorController } from './aggregator/aggregator.controller'
import { AggregatorService } from './aggregator/aggregator.service'
import { DataModule } from './data/data.module'

@Module({
  imports: [ChainModule, AdapterModule, FeedModule, AggregatorModule, DataModule],
  controllers: [AppController, AdapterController, FeedController, AggregatorController],
  providers: [AppService, PrismaService, AdapterService, FeedService, AggregatorService]
})
export class AppModule {}
