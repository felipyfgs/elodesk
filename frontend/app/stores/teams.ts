import { defineStore } from 'pinia'

export interface Team {
  id: string
  accountId: string
  name: string
  description: string | null
  allowAutoAssign: boolean
  createdAt: string
  updatedAt: string
}

export interface TeamMember {
  id: string
  teamId: string
  userId: string
  createdAt: string
}

export const useTeamsStore = defineStore('teams', {
  state: () => ({
    list: [] as Team[],
    membersByTeam: {} as Record<string, TeamMember[]>,
    loading: false
  }),
  getters: {
    byId(): (id: string) => Team | undefined {
      return (id: string) => this.list.find(t => t.id === id)
    }
  },
  actions: {
    setAll(list: Team[]) {
      this.list = list
    },
    upsert(team: Team) {
      const idx = this.list.findIndex(t => t.id === team.id)
      if (idx >= 0) this.list[idx] = team
      else this.list.push(team)
    },
    remove(id: string) {
      this.list = this.list.filter(t => t.id !== id)
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete this.membersByTeam[id]
    },
    setMembers(teamId: string, members: TeamMember[]) {
      this.membersByTeam[teamId] = members
    }
  }
})
