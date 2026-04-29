import { useConversationsStore } from '~/stores/conversations'

// Deep-link support: routes like /conversations/inbox/:id, /conversations/label/:name
// preload the matching scope filter into state. The popover-based filter UI no
// longer changes the URL, but these routes remain valid as bookmarks.
export default defineNuxtRouteMiddleware((to) => {
  const convs = useConversationsStore()
  const path = to.path

  // `conversationType` (Não atendidas) virou preferência persistida em
  // localStorage — ele NÃO é resetado na navegação geral. O deep-link
  // /conversations/unattended ainda pode ligar ele explicitamente abaixo,
  // mas trocar de rota não desliga.
  const clearScope = {
    inboxIds: undefined,
    labelIds: undefined,
    teamIds: undefined
  }

  if (path.includes('/conversations/inbox/')) {
    const inboxId = to.params.id as string
    if (!inboxId) return
    convs.setFilters({ ...clearScope, inboxIds: [inboxId] })
  } else if (path.includes('/conversations/label/')) {
    const labelName = to.params.name as string
    if (!labelName) return
    convs.setFilters({ ...clearScope, labelIds: [labelName] })
  } else if (path.includes('/conversations/team/')) {
    const teamId = to.params.id as string
    if (!teamId) return
    convs.setFilters({ ...clearScope, teamIds: [teamId] })
  } else if (path.includes('/conversations/filter/')) {
    convs.setFilters({ ...clearScope })
  } else if (path.endsWith('/conversations/unattended')) {
    convs.setFilters({ ...clearScope, conversationType: 'unattended' })
  } else if (path.endsWith('/conversations')) {
    convs.setFilters({ ...clearScope })
  }
})
