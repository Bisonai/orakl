import { ValidationPipe } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { NestFactory } from '@nestjs/core'
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger'
import { AppModule } from './app.module'
import { setAppSettings } from './app.settings'

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unreachable code error
BigInt.prototype.toJSON = function (): string {
  return this.toString()
}

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)
  app.useGlobalPipes(new ValidationPipe({ whitelist: false, transform: true }))

  const version = '1.0'
  const config = new DocumentBuilder()
    .setTitle('Orakl L2 API')
    .setDescription('The Orakl L2 Config API description')
    .setVersion(version)
    .build()
  const document = SwaggerModule.createDocument(app, config)
  SwaggerModule.setup('docs', app, document)

  const configService = app.get(ConfigService)
  const port = configService.get('APP_PORT')
  app.enableShutdownHooks()
  await app.listen(port)
}
bootstrap()
