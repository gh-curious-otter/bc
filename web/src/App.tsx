import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ErrorBoundary } from './components/ErrorBoundary';
import { Dashboard } from './views/Dashboard';
import { Agents } from './views/Agents';
import { Channels } from './views/Channels';
import { Costs } from './views/Costs';

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
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
