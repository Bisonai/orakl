// https://www.prisma.io/docs/reference/api-reference/error-reference
export const PRISMA_ERRORS = {
  P2002: (meta) => `Unique constraint failed on the ${meta.target}`
}
