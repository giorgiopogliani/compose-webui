<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const props = defineProps<{
  state: string
  health?: string
}>()

const color = computed(() => {
  const s = props.state.toLowerCase()
  if (s === 'running' || s === 'running (healthy)' || s === 'up') return 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/30'
  if (s === 'paused' || s === 'restarting' || props.health === 'starting')
    return 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/30'
  if (s === 'exited' || s === 'dead' || s === 'stopped' || s === 'removing' || props.health === 'unhealthy')
    return 'bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/30'
  return 'bg-muted text-muted-foreground border-border'
})

const label = computed(() => {
  const s = props.state.toLowerCase()
  if (s === 'running') return props.health && props.health !== '' && props.health !== 'none' ? `running · ${props.health}` : 'running'
  return props.state
})
</script>

<template>
  <Badge variant="outline" :class="cn('font-mono text-xs normal-case', color)">
    {{ label }}
  </Badge>
</template>