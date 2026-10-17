import { Handle, Position, type NodeProps } from '@xyflow/react'
import type { AgentStepFlowNode } from '../types'

const toolIcons: Record<string, string> = {
  bash: '❯',
  read_file: '▤',
  write_file: '✎',
  edit_file: '✎',
  search: '⌕',
  browser: '◎',
  fetch: '⇅',
}

const defaultIcon = '⚙'

export default function AgentStepNode({ data }: NodeProps<AgentStepFlowNode>) {
  const isError = data.status === 'error'
  const icon = data.toolCalled ? (toolIcons[data.toolCalled] ?? defaultIcon) : null

  return (
    <div
      className={`w-64 rounded-lg bg-white shadow-md ${isError ? 'border-2 border-red-500' : 'border-2 border-blue-500'}`}
    >
      <Handle type="target" position={Position.Top} className="!bg-gray-400" />

      <div className="flex items-center justify-between border-b border-gray-100 px-3 py-1.5">
        <span className="text-xs font-semibold text-gray-700">Step {data.stepNumber}</span>
        <span className="text-xs tabular-nums text-gray-500">{data.latencyMs} ms</span>
      </div>

      <div className="px-3 py-2">
        {data.toolCalled && (
          <div className="mb-1 flex items-center gap-1.5">
            <span aria-hidden className="text-sm text-blue-600">
              {icon}
            </span>
            <span className="truncate text-sm font-medium text-gray-900">{data.toolCalled}</span>
            {isError && <span className="ml-auto text-[10px] font-semibold uppercase text-red-500">error</span>}
          </div>
        )}
        <p className="line-clamp-2 text-xs leading-snug text-gray-500">{data.thought}</p>
      </div>

      <div className="border-t border-gray-100 px-3 py-1 text-[10px] tabular-nums text-gray-400">
        {data.tokensUsed} tokens
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-gray-400" />
    </div>
  )
}
