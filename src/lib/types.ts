export interface ServiceStatus {
  name: string
  image: string
  state: string
  health: string
  ports: string
  status: string
}

export interface Project {
  name: string
  status: string
  configFiles: string[]
  workingDir: string
  services: ServiceStatus[]
}

export interface ProjectsResponse {
  projects: Project[]
  error?: string
}

export interface ActionResponse {
  ok: boolean
  message: string
}