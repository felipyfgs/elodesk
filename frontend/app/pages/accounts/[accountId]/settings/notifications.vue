<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })

const route = useRoute()
await navigateTo(`/accounts/${route.params.accountId}/settings/profile`, { replace: true })
</script>
