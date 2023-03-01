import { NestFactory } from '@nestjs/core'
import { AppModule } from './app.module'
import { setAppSettings } from './app.settings'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)
  await app.listen(3000)
}
bootstrap()
