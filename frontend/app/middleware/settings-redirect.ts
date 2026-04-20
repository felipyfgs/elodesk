export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/settings' || to.path === '/settings/') {
    return navigateTo('/settings/profile')
  }
})
