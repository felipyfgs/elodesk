const allowed = ['agents', 'inboxes', 'teams', 'labels']

export default defineNuxtRouteMiddleware((to) => {
  const entity = to.params.entity
  if (typeof entity !== 'string' || !allowed.includes(entity)) {
    throw createError({ statusCode: 404, statusMessage: 'unknown entity' })
  }
})
