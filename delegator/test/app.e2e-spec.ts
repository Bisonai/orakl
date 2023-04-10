import { Test, TestingModule } from '@nestjs/testing'
import { INestApplication } from '@nestjs/common'
import request from 'supertest'
import { AppModule } from './../src/app.module'
import { setAppSettings } from './../src/app.settings'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from './../src/prisma.service'

describe('AppController (e2e)', () => {
  let app: INestApplication
  let prisma
  beforeEach(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule],
      providers: [PrismaService]
    }).compile()

    app = moduleFixture.createNestApplication()
    prisma = moduleFixture.get<PrismaClient>(PrismaService)
    setAppSettings(app)
    await app.init()
  })

  it('/api/v1 (GET)', () => {
    return request(app.getHttpServer()).get('/api/v1').expect(200).expect('Orakl Network Delegator')
  })
})
