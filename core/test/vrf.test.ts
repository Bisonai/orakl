import { describe, expect, beforeEach, test } from '@jest/globals'
import { prove } from '../src/vrf/index'

describe('VRF', function () {
  let alpha_string
  let sk
  beforeEach(() => {
    alpha_string = '123'
    sk = '1c312867cfa4c80e5569faebdd26aa614d700a93f7934c6e34ab510c604014be'
  })

  test('hello', async function () {
    // off-chain
    // const keypair = keygen()
    // console.log('sk', keypair.secret_key)
    // console.log('pk', keypair.public_key.key)
    const proof = prove(sk, alpha_string)
    expect(proof).toStrictEqual(
      '03f4ee74e8c92a46eb2dfb6f1121dcabdb45ff166df38c82aa156bc16edfa6c96907fff897464aeef0a864b2b56056bfe37ffed054f7a132192ee695c02784986b1e4a950b1fc2bd2b334fe28e23c0e696'
    )
    // console.log('proof', proof)
    // console.log('proof.length', proof.length)
    // const [Gamma, c, s] = decode(proof)
    //
    // const fast = getFastVerifyComponents(keypair.public_key.key, proof, alpha)
    //
    // const [status, beta] = verify(keypair.public_key.key, proof, alpha)
    // console.log('beta', beta)
    // console.log('beta.length', beta.length) // 64
  })
})
