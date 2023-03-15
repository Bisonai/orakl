import { Test, TestingModule } from '@nestjs/testing'
import { VrfService } from './vrf.service'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from '../prisma.service'

describe('VrfService', () => {
  let vrf: VrfService
  let chain: ChainService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [VrfService, ChainService, PrismaService]
    }).compile()

    vrf = module.get<VrfService>(VrfService)
    chain = module.get<ChainService>(ChainService)
  })

  it('should be defined', () => {
    expect(vrf).toBeDefined()
  })

  it('should be defined', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'listener-test-chain' })

    // VRF Key
    const vrfKeyObj = await vrf.create({
      sk: 'ebeb5229570725793797e30a426d7ef8aca79d38ff330d7d1f28485d2366de32',
      pk: '045b8175cfb6e7d479682a50b19241671906f706bd71e30d7e80fd5ff522c41bf0588735865a5faa121c3801b0b0581440bdde24b03dc4c4541df9555d15223e82',
      pkX: '41389205596727393921445837404963099032198113370266717620546075917307049417712',
      pkY: '40042424443779217635966540867474786311411229770852010943594459290130507251330',
      keyHash: '0x6f32373625e3d1f8f303196cbb78020ac2503acd1129e44b36b425781a9664ac',
      chain: chainObj.name
    })

    // Cleanup
    await vrf.remove({ id: vrfKeyObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
