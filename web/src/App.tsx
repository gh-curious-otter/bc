import { BrowserRouter, Routes, Route } from 'react-router-dom';
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
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
