import { Module } from '@nestjs/common'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { ConfigService } from '@nestjs/config'
import { FeedModule } from './feed/feed.module'
import { ChainModule } from './chain/chain.module'
import { AdapterModule } from './adapter/adapter.module'
import { AggregatorModule } from './aggregator/aggregator.module'
import { DataModule } from './data/data.module'
import { AggregateModule } from './aggregate/aggregate.module'
import { ServiceModule } from './service/service.module'
import { ListenerModule } from './listener/listener.module'
import { VrfModule } from './vrf/vrf.module'

@Module({
  imports: [
    ChainModule,
    AdapterModule,
    FeedModule,
    AggregatorModule,
    DataModule,
    AggregateModule,
    ServiceModule,
    ListenerModule,
    VrfModule
  ],
  controllers: [AppController],
  providers: [AppService, ConfigService]
})
export class AppModule {}
