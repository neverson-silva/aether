import { redirect } from '@tanstack/react-router'
import { apiGet } from '../api/client'
import type { Me } from '../api/types'
import { useAuthStore } from '../stores/auth'

export async function requireAuth(): Promise<Me> {
  const { user } = useAuthStore.getState()
  if (user) {
    return user
  }
  try {
    const me = await apiGet<Me>('/api/v1/me')
    useAuthStore.getState().setUser(me)
    return me
  } catch {
    throw redirect({ to: '/login' })
  }
}
