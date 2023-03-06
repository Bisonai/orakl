import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { SignModule } from './sign/sign.module';

@Module({
  imports: [SignModule],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
