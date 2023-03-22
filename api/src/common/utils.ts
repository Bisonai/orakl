import { HttpStatus, HttpException, Logger } from '@nestjs/common'

/**
 * Find chain given `chainName`.
 *
 * @params {} prisma client for chain
 * @params {string} chain name
 * @params {Logger} NestJS logger
 * @return {} chain object represented by a `chainName`
 * @exception {HttpException} raise when there is no chain with `chainName`
 */
export async function getChain({
  chain,
  chainName,
  logger
}: {
  chain
  chainName: string
  logger: Logger
}) {
  const chainObj = await chain.findUnique({
    where: { name: chainName }
  })

  if (chainObj == null) {
    const msg = `chain.name [${chainName}] not found`
    logger.error(msg)
    throw new HttpException(msg, HttpStatus.NOT_FOUND)
  }

  return chainObj
}

/**
 * Find service given `serviceName`.
 *
 * @params {} prisma client for service
 * @params {string} service name
 * @params {Logger} NestJS logger
 * @return {} service object represented by a `serviceName`
 * @exception {HttpException} raise when there is no service with `serviceName`
 */
export async function getService({
  service,
  serviceName,
  logger
}: {
  service
  serviceName: string
  logger: Logger
}) {
  const serviceObj = await service.findUnique({
    where: { name: serviceName }
  })

  if (serviceObj == null) {
    const msg = `service.name [${serviceName}] not found`
    logger.error(msg)
    throw new HttpException(msg, HttpStatus.NOT_FOUND)
  }

  return serviceObj
}
