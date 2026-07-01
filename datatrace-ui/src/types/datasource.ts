export interface CredentialOption {
  id: string
  name: string
  username: string
  description?: string
}

export interface DataSourceRecord {
  id: string
  name: string
  type: 'mysql' | 'elasticsearch' | string
  host: string
  port: number
  username?: string
  database_name?: string
  credential_id?: string | null
}

export interface DataSourceFormData {
  id?: string
  name: string
  type: 'mysql' | 'elasticsearch'
  host: string
  port: number | null
  auth_type: 'manual' | 'credential'
  credential_id: string
  username: string
  password: string
  database_name: string
}

export interface DataSourcePayload {
  name: string
  type: string
  host: string
  port: number
  credential_id?: string
  username?: string
  password?: string
  database_name?: string
}

export interface TestConnectionResult {
  success: boolean
  version: string
  message: string
}

export interface DataSourceHealthStatus {
  id: string
  status: 'testing' | 'success' | 'failed' | string
  message: string
  timestamp: string
}
