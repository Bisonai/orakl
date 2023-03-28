import { NestFactory } from '@nestjs/core'
import { AppModule } from './app.module'
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger'
import { setAppSettings } from './app.settings'
import { ConfigService } from '@nestjs/config'

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unreachable code error
BigInt.prototype.toJSON = function (): string {
  return this.toString()
}

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)

  const config = new DocumentBuilder()
    .setTitle('Orakl Network Delegator')
    .setDescription('The Orakl Network Delegator description')
    .setVersion('1.0')
    .build()
  const document = SwaggerModule.createDocument(app, config)
  SwaggerModule.setup('docs', app, document)

  const configService = app.get(ConfigService)
  const port = configService.get('APP_PORT')
  await app.listen(port)
}
bootstrap()
