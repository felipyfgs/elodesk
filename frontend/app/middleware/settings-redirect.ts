export default defineNuxtRouteMiddleware((to) => {
  const match = to.path.match(/^\/accounts\/([^/]+)\/settings\/?$/)
  if (match) {
    return navigateTo(`/accounts/${match[1]}/settings/account`)
  }
})
