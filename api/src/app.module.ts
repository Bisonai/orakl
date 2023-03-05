import { Module } from '@nestjs/common'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { FeedModule } from './feed/feed.module'
import { ChainModule } from './chain/chain.module'
import { AdapterModule } from './adapter/adapter.module'
import { AggregatorModule } from './aggregator/aggregator.module'
import { DataModule } from './data/data.module'

@Module({
  imports: [ChainModule, AdapterModule, FeedModule, AggregatorModule, DataModule],
  controllers: [AppController],
  providers: [AppService]
})
export class AppModule {}
