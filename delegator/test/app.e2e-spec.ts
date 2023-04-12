import { Test, TestingModule } from '@nestjs/testing'
import { INestApplication } from '@nestjs/common'
import request from 'supertest'
import { AppModule } from './../src/app.module'
import { setAppSettings } from './../src/app.settings'

describe('AppController (e2e)', () => {
  let app: INestApplication
  beforeEach(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule]
    }).compile()

    app = moduleFixture.createNestApplication()
    setAppSettings(app)
    await app.init()
  })

  it('/api/v1 (GET)', () => {
    return request(app.getHttpServer()).get('/api/v1').expect(200).expect('Orakl Network Delegator')
  })
})
