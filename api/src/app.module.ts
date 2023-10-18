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
import { ReporterModule } from './reporter/reporter.module'
import { ErrorModule } from './error/error.module'
import { ProxyModule } from './proxy/proxy.module'
import { L2aggregatorModule } from './l2aggregator/l2aggregator.module';

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
