import { Test, TestingModule } from '@nestjs/testing'
import { INestApplication } from '@nestjs/common'
import * as request from 'supertest'
import { AppModule } from './../src/app.module'

describe('AppController (e2e)', () => {
  let app: INestApplication

  beforeEach(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule]
    }).compile()

    app = moduleFixture.createNestApplication()
    await app.init()
  })

  it('/health (GET)', () => {
    return request(app.getHttpServer()).get('/health').expect(200).expect('OK')
  })

  it('/api (GET)', () => {
    return request(app.getHttpServer()).get('/api').expect(200).expect('Orakl Network Fetcher')
  })
})
