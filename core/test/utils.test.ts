import { describe, expect, test } from '@jest/globals'
import axios from 'axios'
import MockAdapter from 'axios-mock-adapter'
import { OraklErrorCode } from '../src/errors'
import { axiosTimeout } from '../src/utils'

describe('Utils', function () {
  test('axiosTimeout should return proper value on reqeusts', async function () {
    const fakeUrl = `https://fake-api.com`
    const mockEndpoint = new MockAdapter(axios)

    mockEndpoint.onGet(fakeUrl).reply(200, { message: 'fake get data' })
    mockEndpoint.onPost(fakeUrl).reply(200, { message: 'fake post data' })
    mockEndpoint.onDelete(fakeUrl).reply(200, { message: 'fake delete data' })
    mockEndpoint.onPatch(fakeUrl).reply(200, { message: 'fake patch data' })

    let r = await axiosTimeout.get(fakeUrl)
    expect(r?.data.message).toEqual('fake get data')

    r = await axiosTimeout.post(fakeUrl, { mock: 'mockData' })
    expect(r?.data.message).toEqual('fake post data')

    r = await axiosTimeout.delete(fakeUrl)
    expect(r?.data.message).toEqual('fake delete data')

    r = await axiosTimeout.patch(fakeUrl)
    expect(r?.data.message).toEqual('fake patch data')
  })

  test('axiosTimeout should handle timeout error', async function () {
    const fakeUrl = `https://fake-api.com`
    const mockEndpoint = new MockAdapter(axios)

    mockEndpoint.onPost(fakeUrl).timeout()
    mockEndpoint.onGet(fakeUrl).timeout()
    mockEndpoint.onDelete(fakeUrl).timeout()
    mockEndpoint.onPatch(fakeUrl).timeout()

    try {
      await axiosTimeout.post(fakeUrl)
    } catch (e) {
      expect(e.code).toEqual(OraklErrorCode.AxiosTimeOut)
    }

    try {
      await axiosTimeout.get(fakeUrl)
    } catch (e) {
      expect(e.code).toEqual(OraklErrorCode.AxiosTimeOut)
    }

    try {
      await axiosTimeout.delete(fakeUrl)
    } catch (e) {
      expect(e.code).toEqual(OraklErrorCode.AxiosTimeOut)
    }

    try {
      await axiosTimeout.patch(fakeUrl)
    } catch (e) {
      expect(e.code).toEqual(OraklErrorCode.AxiosTimeOut)
    }
  })
})
