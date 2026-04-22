import { useConversationsStore } from '~/stores/conversations'

export default defineNuxtRouteMiddleware((to) => {
  const convs = useConversationsStore()
  const path = to.path

  if (path.includes('/conversations/inbox/')) {
    const inboxId = to.params.id as string
    convs.setFilters({ inboxId })
  } else if (path.includes('/conversations/label/')) {
    const labelName = to.params.name as string
    convs.setFilters({ labelId: labelName })
  } else if (path.includes('/conversations/team/')) {
    const teamId = to.params.id as string
    convs.setFilters({ teamId })
  } else if (path.includes('/conversations/filter/')) {
    convs.setFilters({})
  } else if (path.endsWith('/conversations/mentions')) {
    convs.setFilters({ tab: 'mentions' })
  } else if (path.endsWith('/conversations/unattended')) {
    convs.setFilters({ tab: 'unassigned' })
  }
})
