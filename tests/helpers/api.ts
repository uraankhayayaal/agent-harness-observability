import { config } from './env'

export interface ApiNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: {
    stepNumber: number
    thought: string
    toolCalled: string
    status: string
    latencyMs: number
    tokensUsed: number
  }
}

export interface ApiEdge {
  id: string
  source: string
  target: string
  animated: boolean
}

export interface ApiTrajectory {
  nodes: ApiNode[]
  edges: ApiEdge[]
}

export async function getTrajectory(runID: string): Promise<ApiTrajectory> {
  const res = await fetch(
    `${config.backendUrl}${config.apiBasePath}/runs/${encodeURIComponent(runID)}/trajectory`,
  )
  if (!res.ok) {
    throw new Error(`API ${res.status}: ${await res.text()}`)
  }
  return (await res.json()) as ApiTrajectory
}

export async function waitForTrajectory(
  runID: string,
  expectedNodes: number,
  timeoutMs = 30_000,
): Promise<ApiTrajectory> {
  const deadline = Date.now() + timeoutMs
  let lastError: unknown
  while (Date.now() < deadline) {
    try {
      const traj = await getTrajectory(runID)
      if (traj.nodes.length >= expectedNodes) return traj
      lastError = new Error(`nodes=${traj.nodes.length}, want ${expectedNodes}`)
    } catch (e) {
      lastError = e
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`trajectory for ${runID} not ready after ${timeoutMs}ms: ${lastError}`)
}

export async function backendHealthy(): Promise<boolean> {
  try {
    const res = await fetch(`${config.backendUrl}/healthz`)
    return res.ok
  } catch {
    return false
  }
}

export async function frontendReachable(): Promise<boolean> {
  try {
    const res = await fetch(config.frontendUrl)
    return res.ok
  } catch {
    return false
  }
}
