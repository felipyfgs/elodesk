<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const id = auth.account?.id
if (id) {
  navigateTo(`/accounts/${id}${route.fullPath}`, { replace: true })
} else {
  navigateTo('/login', { replace: true })
}
</script>
