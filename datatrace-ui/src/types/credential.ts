export interface CredentialRecord {
  id: string
  name: string
  username: string
  description?: string
  created_at: string
}

export interface CreateCredentialRequest {
  name: string
  username: string
  password?: string
  description?: string
}
