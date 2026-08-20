<script setup lang="ts">
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { RefreshCw, Rocket, FolderOpen, Play } from 'lucide-vue-next'
import type { Project } from '@/lib/types'
import StatusBadge from './StatusBadge.vue'

defineProps<{
  project: Project
  pending: string[]
}>()

const emit = defineEmits<{
  action: [payload: { action: 'up' | 'restart'; service?: string }]
}>()

const key = (project: string, service?: string) => (service ? `${project}/${service}` : project)
</script>

<template>
  <Card class="overflow-hidden">
    <CardHeader class="flex flex-row items-start justify-between gap-4 space-y-0">
      <div class="min-w-0 space-y-1">
        <CardTitle class="flex items-center gap-2">
          <span class="truncate font-mono text-base">{{ project.name }}</span>
          <StatusBadge :state="project.status" />
        </CardTitle>
        <CardDescription class="flex flex-col gap-0.5 text-xs">
          <span class="flex items-center gap-1 truncate font-mono">
            <FolderOpen class="h-3 w-3 shrink-0" />
            {{ project.workingDir || project.configFiles.join(', ') || '—' }}
          </span>
        </CardDescription>
      </div>
      <div class="flex shrink-0 gap-2">
        <Button
          size="sm"
          variant="outline"
          :disabled="pending.includes(key(project.name))"
          @click="emit('action', { action: 'restart' })"
        >
          <RefreshCw class="h-3.5 w-3.5" />
          Restart
        </Button>
        <Button
          size="sm"
          :disabled="pending.includes(key(project.name))"
          @click="emit('action', { action: 'up' })"
        >
          <Rocket class="h-3.5 w-3.5" />
          Up
        </Button>
      </div>
    </CardHeader>
    <CardContent>
      <Separator class="mb-3" />
      <div v-if="project.services.length === 0" class="py-2 text-sm text-muted-foreground">
        No services found.
      </div>
      <ul v-else class="space-y-1.5">
        <li
          v-for="svc in project.services"
          :key="svc.name"
          class="flex items-center justify-between gap-3 rounded-md border px-3 py-2"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="truncate font-mono text-sm font-medium">{{ svc.name }}</span>
              <StatusBadge :state="svc.state" :health="svc.health" />
            </div>
            <div class="truncate text-xs text-muted-foreground">
              {{ svc.image && svc.image !== svc.name ? svc.image : '' }}
              {{ svc.ports ? ` · ${svc.ports}` : '' }}
            </div>
          </div>
          <div class="flex shrink-0 gap-1.5">
            <Button
              size="sm"
              variant="ghost"
              :disabled="pending.includes(key(project.name, svc.name))"
              @click="emit('action', { action: 'restart', service: svc.name })"
            >
              <RefreshCw class="h-3.5 w-3.5" />
              Restart
            </Button>
            <Button
              size="sm"
              variant="outline"
              :disabled="pending.includes(key(project.name, svc.name))"
              @click="emit('action', { action: 'up', service: svc.name })"
            >
              <Play class="h-3.5 w-3.5" />
              Up
            </Button>
          </div>
        </li>
      </ul>
    </CardContent>
  </Card>
</template>