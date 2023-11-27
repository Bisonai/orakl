export interface IVrfConfig {
  sk: string
  pk: string
  pkX: string
  pkY: string
}

export interface IVrfResponse {
  pk: [string, string]
  proof: [string, string, string, string]
  uPoint: [string, string]
  vComponents: [string, string, string, string]
}
