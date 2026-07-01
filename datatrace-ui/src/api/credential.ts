import { http } from './http'
import type { CredentialOption } from '../types/datasource'
import type { CreateCredentialRequest, CredentialRecord } from '../types/credential'

export const credentialApi = {
  list() {
    return http.get<CredentialRecord[]>('/api/v1/credentials')
  },
  getById(id: string) {
    return http.get<CredentialRecord>(`/api/v1/credentials/${id}`)
  },
  create(payload: CreateCredentialRequest) {
    return http.post<CredentialRecord>('/api/v1/credentials', payload)
  },
  update(id: string, payload: CreateCredentialRequest) {
    return http.put<CredentialRecord>(`/api/v1/credentials/${id}`, payload)
  },
  remove(id: string) {
    return http.delete<null>(`/api/v1/credentials/${id}`)
  }
}
