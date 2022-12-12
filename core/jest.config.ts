import type { Config } from 'jest'

const config: Config = {
  verbose: true,
  moduleFileExtensions: ['js', 'json', 'ts'],
  rootDir: '.',
  testMatch: ['<rootDir>/test/*.ts'],
  moduleDirectories: ['node_modules', 'dist/src/', '<rootDir>'],
  testPathIgnorePatterns: [],
  transform: {
    '^.+\\.(t|j)s$': 'ts-jest'
  },
  transformIgnorePatterns: ['node_modules/(?!@bisonai)'],
  extensionsToTreatAsEsm: ['.ts'],
  collectCoverageFrom: ['**/*.(t|j)s'],
  coverageDirectory: '../coverage',
  testEnvironment: 'node',
  maxConcurrency: 1,
  maxWorkers: 1,
  bail: true
}

export default config
