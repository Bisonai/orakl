import { NestFactory } from '@nestjs/core'
import { AppModule } from './app.module'
import { setAppSettings } from './app.settings'
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)

  const version = '1.0'
  const config = new DocumentBuilder()
    .setTitle('Orakl Network API')
    .setDescription('The Orakl Network API description')
    .setVersion(version)
    .build()
  const document = SwaggerModule.createDocument(app, config)
  SwaggerModule.setup('docs', app, document)

  await app.listen(3000)
}
bootstrap()
