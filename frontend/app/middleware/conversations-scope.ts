import { useConversationsStore } from '~/stores/conversations'

export default defineNuxtRouteMiddleware((to) => {
  const convs = useConversationsStore()

  // Route-level filter injection
  const path = to.path

  if (path.startsWith('/conversations/inbox/')) {
    const inboxId = to.params.id as string
    convs.setFilters({ inboxId })
  } else if (path.startsWith('/conversations/label/')) {
    const labelName = to.params.name as string
    convs.setFilters({ labelId: labelName })
  } else if (path.startsWith('/conversations/team/')) {
    const teamId = to.params.id as string
    convs.setFilters({ teamId })
  } else if (path.startsWith('/conversations/filter/')) {
    const _filterId = to.params.id as string
    // Saved filter — would need integration with savedFilters store
    convs.setFilters({})
  } else if (path === '/conversations/mentions') {
    convs.setFilters({ tab: 'mentions' })
  } else if (path === '/conversations/unattended') {
    convs.setFilters({ tab: 'unassigned' })
  }
})
