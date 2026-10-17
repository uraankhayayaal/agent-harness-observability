import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { fetchTrajectory } from './api/client'
import TrajectoryGraph from './components/TrajectoryGraph'
import type { Trajectory } from './types'

export default function App() {
  const [input, setInput] = useState(
    () => new URLSearchParams(window.location.search).get('run') ?? '',
  )
  const [trajectory, setTrajectory] = useState<Trajectory | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async (id: string) => {
    const runId = id.trim()
    if (!runId) {
      setError('Введите run_id')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await fetchTrajectory(runId)
      setTrajectory(data)
      const url = new URL(window.location.href)
      url.searchParams.set('run', runId)
      window.history.replaceState({}, '', url)
    } catch (e) {
      setTrajectory(null)
      setError(e instanceof Error ? e.message : 'Не удалось загрузить траекторию')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    const initial = new URLSearchParams(window.location.search).get('run')
    if (initial) {
      void load(initial)
    }
  }, [load])

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    void load(input)
  }

  const totalTokens = trajectory?.nodes.reduce((acc, n) => acc + n.data.tokensUsed, 0) ?? 0
  const totalLatency = trajectory?.nodes.reduce((acc, n) => acc + n.data.latencyMs, 0) ?? 0
  const errorSteps = trajectory?.nodes.filter((n) => n.data.status === 'error').length ?? 0

  return (
    <div className="flex h-full flex-col bg-gray-50">
      <header className="border-b border-gray-200 bg-white px-6 py-3">
        <div className="flex flex-wrap items-center gap-4">
          <h1 className="text-sm font-semibold tracking-tight text-gray-900">
            Agent Harness Observability
          </h1>

          <form onSubmit={onSubmit} className="flex flex-1 items-center gap-2">
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="run_id (uuid)"
              className="w-full max-w-md rounded-md border border-gray-300 px-3 py-1.5 font-mono text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:outline-none"
            />
            <button
              type="submit"
              disabled={loading}
              className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Загрузка…' : 'Показать'}
            </button>
          </form>

          {trajectory && (
            <div className="flex items-center gap-3 text-xs tabular-nums text-gray-500">
              <span>{trajectory.nodes.length} шагов</span>
              <span>{totalTokens} токенов</span>
              <span>{totalLatency} ms</span>
              <span className={errorSteps > 0 ? 'font-semibold text-red-500' : ''}>
                {errorSteps} ошибок
              </span>
            </div>
          )}
        </div>

        {error && (
          <p className="mt-2 text-xs font-medium text-red-500" role="alert">
            {error}
          </p>
        )}
      </header>

      <main className="min-h-0 flex-1">
        {trajectory ? (
          <TrajectoryGraph nodes={trajectory.nodes} edges={trajectory.edges} />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-gray-400">
            Введите run_id, чтобы увидеть траекторию агента
          </div>
        )}
      </main>
    </div>
  )
}
