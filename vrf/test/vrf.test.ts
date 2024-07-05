import { beforeEach, describe, expect, test } from '@jest/globals'
import { getFastVerifyComponents, prove, verify } from '../src/index'

// The following tests are used to make sure that the original
// implementation was not affected during refactoring.
// TODO Add tests from https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-vrf-10#appendix-A-1

describe('VRF', function () {
  let alpha
  let SK
  let PK
  beforeEach(() => {
    alpha = '123'
    SK = '927a5ce18fd9bba52e8a006633600abf1cdfbaa87b76401590945cd211142688'
    PK =
      '04c322317773795c4c1c1f1dc57a0351af5c9cc5d6e7e46a5e477fa31f482ff4edf19814af61fa8baf8d70280171d1163bdcd732b43953aa02f545d8c4b4b5b19c'
  })

  test('Test proof generation', function () {
    const proofGt =
      '03f2e53ed55d152362e73459aeb64607a90900bac77fbc807955d7dcef835a0a2944a392fa7ce96e80ef347bc30e9deb8b858d6bfff20af8828635a9994ceffb90a17c40807dbe13c2056d87fcead9e634'
    const proof = prove(SK, alpha)
    expect(proof).toStrictEqual(proofGt)
  })

  test('Test getFastVerifyComponents', function () {
    const fastGt = {
      uX: '25913720932606563440475568391069963563581295559396919082558110777218391807540',
      uY: '1202527622214681382476918210290626287348734880826780179410055860676922606226',
      sHX: '2439383153493978275097713356471866358237926157287722820098450836881483891304',
      sHY: '108088785076600261524120804093472180727178271089753393720973481338416338814137',
      cGX: '45091331714886762885956250490260438795398807115167088791637934171464089638804',
      cGY: '41076761133918350571240208725724766016199597466412944191921220776340643881905',
    }

    const proof = prove(SK, alpha)
    const fast = getFastVerifyComponents(PK, proof, alpha)

    expect(fast).toStrictEqual(fastGt)
  })

  test('Test verify', function () {
    const statusGt = 'VALID'
    const betaGt = 'd530051e07609707944aa2f5cdf0cdd81f24d28fa283dae2526b46e31d0426a8'

    const proof = prove(SK, alpha)
    const [status, beta] = verify(PK, proof, alpha)

    expect(status).toStrictEqual(statusGt)
    expect(beta).toStrictEqual(betaGt)
  })
})
