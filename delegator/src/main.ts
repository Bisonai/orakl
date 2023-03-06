import { NestFactory } from '@nestjs/core'
import { AppModule } from './app.module'
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger'
import { setAppSettings } from './app.settings'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  setAppSettings(app)

  const config = new DocumentBuilder()
    .setTitle('Orakl Network Delegator')
    .setDescription('The Orakl Network Delegator API description')
    .setVersion('1.0')
    .build()
  const document = SwaggerModule.createDocument(app, config)
  SwaggerModule.setup('docs', app, document)

  await app.listen(3000)
}
bootstrap()
