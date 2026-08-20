<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { AlertTriangle, Moon, RefreshCw, Sun, Wrench } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchProjects, runAction } from '@/lib/api'
import type { Project } from '@/lib/types'
import ProjectCard from './components/ProjectCard.vue'

const projects = ref<Project[]>([])
const error = ref('')
const loading = ref(true)
const actionError = ref('')
const autoRefresh = ref(true)
const pending = ref<string[]>([])
const lastUpdated = ref('')
let timer: ReturnType<typeof setInterval> | undefined

const isDark = ref(document.documentElement.classList.contains('dark'))

const toggleTheme = () => {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

const refresh = async () => {
  loading.value = true
  actionError.value = ''
  try {
    const res = await fetchProjects()
    projects.value = res.projects
    error.value = res.error ?? ''
    lastUpdated.value = new Date().toLocaleTimeString()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

const toggleAutoRefresh = () => {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) startTimer()
  else stopTimer()
}

const startTimer = () => {
  stopTimer()
  timer = setInterval(refresh, 5000)
}

const stopTimer = () => {
  if (timer) {
    clearInterval(timer)
    timer = undefined
  }
}

const handleAction = async (payload: { action: 'up' | 'restart'; service?: string }) => {
  const key = (p: Project) =>
    payload.service ? `${p.name}/${payload.service}` : p.name
  const target = projects.value.find((p) =>
    payload.service ? p.services.some((s) => s.name === payload.service) : true,
  )
  if (!target) return
  pending.value = [...pending.value, key(target)]
  actionError.value = ''
  try {
    const res = await runAction(payload.action, target.name, payload.service)
    if (!res.ok) actionError.value = `${payload.action} ${target.name}${payload.service ? `/${payload.service}` : ''}: ${res.message}`
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  } finally {
    pending.value = pending.value.filter((k) => k !== key(target))
    await refresh()
  }
}

const totalServices = computed(() =>
  projects.value.reduce((n, p) => n + p.services.length, 0),
)

const runningServices = computed(
  () =>
    projects.value.reduce(
      (n, p) => n + p.services.filter((s) => s.state === 'running').length,
      0,
    ),
)

onMounted(() => {
  refresh()
  startTimer()
})

onBeforeUnmount(stopTimer)
</script>

<template>
  <div class="min-h-screen">
    <header class="sticky top-0 z-10 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-6 py-4">
        <div class="flex items-center gap-2.5">
          <div class="flex h-8 w-8 items-center justify-center rounded-md border bg-muted">
            <Wrench class="h-4 w-4" />
          </div>
          <div>
            <h1 class="text-lg font-semibold leading-tight">Compose Dashboard</h1>
            <p class="text-xs text-muted-foreground">
              {{ projects.length }} projects · {{ runningServices }}/{{ totalServices }} services running
            </p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Button size="icon" variant="ghost" :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'" @click="toggleTheme">
            <Sun v-if="isDark" class="h-4 w-4" />
            <Moon v-else class="h-4 w-4" />
          </Button>
          <Button size="sm" variant="outline" @click="toggleAutoRefresh">
            {{ autoRefresh ? 'Auto-refresh: on' : 'Auto-refresh: off' }}
          </Button>
          <Button size="sm" :disabled="loading" @click="refresh">
            <RefreshCw class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" />
            Refresh
          </Button>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-6 py-6">
      <div v-if="lastUpdated" class="mb-4 text-xs text-muted-foreground">
        Updated {{ lastUpdated }}
      </div>

      <div v-if="error" class="mb-4 flex items-start gap-2 rounded-md border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400">
        <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
        <div>
          <p class="font-medium">Cannot reach Docker</p>
          <p class="font-mono text-xs">{{ error }}</p>
        </div>
      </div>

      <div v-if="actionError" class="mb-4 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-600 dark:text-amber-400">
        {{ actionError }}
      </div>

      <div v-if="loading && projects.length === 0" class="grid gap-4">
        <Skeleton v-for="i in 3" :key="i" class="h-32 w-full" />
      </div>

      <div v-else-if="projects.length === 0 && !error" class="py-16 text-center text-muted-foreground">
        No compose projects found.
      </div>

      <div v-else class="grid gap-4">
        <ProjectCard
          v-for="p in projects"
          :key="p.name"
          :project="p"
          :pending="pending"
          @action="handleAction"
        />
      </div>
    </main>
  </div>
</template>