import { Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { AdapterModule } from './adapter/adapter.module'
import { AggregateModule } from './aggregate/aggregate.module'
import { AggregatorModule } from './aggregator/aggregator.module'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { ChainModule } from './chain/chain.module'
import { DataModule } from './data/data.module'
import { ErrorModule } from './error/error.module'
import { FeedModule } from './feed/feed.module'
import { L2aggregatorModule } from './l2aggregator/L2aggregator.module'
import { ListenerModule } from './listener/listener.module'
import { ProxyModule } from './proxy/proxy.module'
import { ReporterModule } from './reporter/reporter.module'
import { ServiceModule } from './service/service.module'
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
    VrfModule,
    ReporterModule,
    ErrorModule,
    ProxyModule,
    L2aggregatorModule
  ],
  controllers: [AppController],
  providers: [AppService, ConfigService]
})
export class AppModule {}
