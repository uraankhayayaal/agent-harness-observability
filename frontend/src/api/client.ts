import type { Trajectory } from '../types'

const apiBasePath = import.meta.env.VITE_API_BASE_PATH || '/v1'

export async function fetchTrajectory(runId: string): Promise<Trajectory> {
  const res = await fetch(`${apiBasePath}/runs/${encodeURIComponent(runId)}/trajectory`)
  if (res.status === 404) {
    throw new Error('Запуск не найден')
  }
  if (!res.ok) {
    throw new Error(`Ошибка сервера: ${res.status}`)
  }
  return (await res.json()) as Trajectory
}
