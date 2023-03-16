import { IVrfConfig } from './types'

export async function getVrfConfig(chain: string) /*: Promise<IVrfConfig>*/ {
  // const query = `SELECT sk, pk, pk_x, pk_y, key_hash FROM VrfKey
  //   INNER JOIN Chain ON Chain.id = VrfKey.chainId AND Chain.name='${chain}'`
  // const vrfConfig = await db.get(query)
  // return vrfConfig
}
