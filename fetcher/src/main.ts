import { ConfigService } from '@nestjs/config'
import { NestFactory } from '@nestjs/core'
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger'
import { AppModule } from './app.module'
import { setAppSettings } from './app.settings'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)

  const version = '1.0'
  const config = new DocumentBuilder()
    .setTitle('Orakl Network Fetcher')
    .setDescription('The Orakl Network Fetcher description')
    .setVersion(version)
    .build()
  const document = SwaggerModule.createDocument(app, config)
  SwaggerModule.setup('docs', app, document)

  const configService = app.get(ConfigService)
  const port = configService.get('APP_PORT')
  await app.listen(port)
}
bootstrap()
