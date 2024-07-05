import { INestApplication } from '@nestjs/common'
import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import * as request from 'supertest'
import { AppModule } from '../src/app.module'
import { setAppSettings } from '../src/app.settings'
import { PrismaService } from '../src/prisma.service'

describe('AppController (e2e)', () => {
  let app: INestApplication
  let prisma

  beforeEach(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule],
      providers: [PrismaService],
    }).compile()

    app = moduleFixture.createNestApplication()
    prisma = moduleFixture.get<PrismaClient>(PrismaService)
    setAppSettings(app)
    await app.init()
  }, 10000)

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
    await app.close()
  })

  it('/api/v1 (GET)', async () => {
    return await request(app.getHttpServer())
      .get('/api/v1')
      .expect(200)
      .expect('Orakl L2 Config Api!')
  })
})
