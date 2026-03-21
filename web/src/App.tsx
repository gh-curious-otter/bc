import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ErrorBoundary } from './components/ErrorBoundary';
import { Dashboard } from './views/Dashboard';
import { Agents } from './views/Agents';
import { Channels } from './views/Channels';
import { Costs } from './views/Costs';
import { Roles } from './views/Roles';
import { Tools } from './views/Tools';
import { MCP } from './views/MCP';
import { Logs } from './views/Logs';
import { Doctor } from './views/Doctor';
import { Cron } from './views/Cron';
import { Secrets } from './views/Secrets';
import { Workspace } from './views/Workspace';

function NotFound() {
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6">
      <p className="text-6xl font-bold font-mono text-bc-muted">404</p>
      <p className="mt-2 text-bc-muted">Page not found</p>
      <Link to="/" className="mt-4 text-sm text-bc-accent hover:underline">Back to Dashboard</Link>
    </div>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<ErrorBoundary><Dashboard /></ErrorBoundary>} />
            <Route path="agents" element={<ErrorBoundary><Agents /></ErrorBoundary>} />
            <Route path="channels" element={<ErrorBoundary><Channels /></ErrorBoundary>} />
            <Route path="costs" element={<ErrorBoundary><Costs /></ErrorBoundary>} />
            <Route path="roles" element={<ErrorBoundary><Roles /></ErrorBoundary>} />
            <Route path="tools" element={<ErrorBoundary><Tools /></ErrorBoundary>} />
            <Route path="mcp" element={<ErrorBoundary><MCP /></ErrorBoundary>} />
            <Route path="logs" element={<ErrorBoundary><Logs /></ErrorBoundary>} />
            <Route path="doctor" element={<ErrorBoundary><Doctor /></ErrorBoundary>} />
            <Route path="cron" element={<ErrorBoundary><Cron /></ErrorBoundary>} />
            <Route path="secrets" element={<ErrorBoundary><Secrets /></ErrorBoundary>} />
            <Route path="workspace" element={<ErrorBoundary><Workspace /></ErrorBoundary>} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
