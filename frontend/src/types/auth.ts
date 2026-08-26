export interface AuthSession {
  authenticated: boolean
  username: string
  role: 'administrator' | 'operator' | 'viewer'
  mustChangePassword: boolean
}

export interface UserAccount {
  id: string
  username: string
  role: AuthSession['role']
  mustChangePassword: boolean
  createdAt: string
  updatedAt: string
}
