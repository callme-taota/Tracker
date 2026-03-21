import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/app-shell'
import Dashboard from '@/pages/Dashboard'
import Pipeline from '@/pages/Pipeline'
import Sources from '@/pages/Sources'
import Items from '@/pages/Items'
import Summaries from '@/pages/Summaries'
import Interests from '@/pages/Interests'
import Plugins from '@/pages/Plugins'
import PluginConfigPage from '@/pages/PluginConfigPage'
import PipelineEditor from '@/pages/PipelineEditor'
import Pipelines from '@/pages/Pipelines'

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/pipelines" element={<Pipelines />} />
        <Route path="/pipelines/:id" element={<PipelineEditor />} />
        <Route path="/pipeline" element={<Pipeline />} />
        <Route path="/sources" element={<Sources />} />
        <Route path="/items" element={<Items />} />
        <Route path="/summaries" element={<Summaries />} />
        <Route path="/interests" element={<Interests />} />
        <Route path="/plugins" element={<Plugins />} />
        <Route path="/plugins/:pluginId" element={<PluginConfigPage />} />
        <Route path="*" element={<Navigate to="/pipelines" replace />} />
      </Route>
    </Routes>
  )
}
