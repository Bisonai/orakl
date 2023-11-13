// /**
//  * Fetch single reporter given its ID from the Orakl Network API.
//  *
//  * @param {string} reporter ID
//  * @param {pino.Logger} logger
//  * @return {IReporterConfig} reporter configuration
//  * @exception {GetReporterRequestFailed}
//  */
// export async function getReporter({
//   id,
//   logger
// }: {
//   id: string
//   logger?: Logger
// }): Promise<IReporterConfig> {
//   //   try {
//   //     const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `reporter/${id}`)
//   //     return (await axios.get(endpoint))?.data
//   //   } catch (e) {
//   //     logger?.error({ name: 'getReporter', file: FILE_NAME, ...e }, 'error')
//   //     throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
//   //   }
// }
