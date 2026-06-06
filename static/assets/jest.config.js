/** @type {import('ts-jest').JestConfigWithTsJest} **/
module.exports = {
preset: "ts-jest",
  testEnvironment: "node",
  transform: {
    "^.+.ts?$": ["ts-jest",{}],
  },
  moduleNameMapper: {
    "../socket": "../socket.ts",
    "./api": "./api.ts",
    "../neith_types": "../neith_types.ts",
    "./neith_types": "./neith_types.ts",
  }
};