import type { JestConfigWithTsJest } from 'ts-jest'

const jestConfig: JestConfigWithTsJest = {
  verbose: true,
  preset: 'ts-jest/presets/default-esm',
  moduleNameMapper: {
    '^(\\.{1,2}/.*)\\.js$': '$1',
  },
  transform: {
    '^.+\\.(t|j)s$': [
      'ts-jest',
      {
        useESM: true,
      },
    ],
  },
  moduleFileExtensions: ['js', 'json', 'ts'],
  rootDir: '.',
  testMatch: ['<rootDir>/test/*.ts'],
  moduleDirectories: ['node_modules', 'dist/src/', '<rootDir>'],
  testPathIgnorePatterns: ['utils.ts'],
  transformIgnorePatterns: ['node_modules/(?!@bisonai)'],
  extensionsToTreatAsEsm: ['.ts'],
  collectCoverageFrom: ['**/*.(t|j)s'],
  coverageDirectory: '../coverage',
  testEnvironment: 'node',
  maxConcurrency: 1,
  maxWorkers: 1,
  bail: true,
}

export default jestConfig
