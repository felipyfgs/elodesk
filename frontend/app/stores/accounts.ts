import { defineStore } from 'pinia'
import type { AuthAccount } from './auth'

export const useAccountsStore = defineStore('accounts', {
  state: () => ({
    memberships: [] as AuthAccount[],
    current: null as AuthAccount | null
  }),
  actions: {
    setMemberships(list: AuthAccount[]) {
      this.memberships = list
      if (!this.current && list.length) this.current = list[0] ?? null
    },
    switch(account: AuthAccount) {
      this.current = account
    }
  }
})
