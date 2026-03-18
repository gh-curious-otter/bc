import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Dashboard } from './views/Dashboard';
import { Agents } from './views/Agents';
import { Channels } from './views/Channels';
import { Costs } from './views/Costs';

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="agents" element={<Agents />} />
          <Route path="channels" element={<Channels />} />
          <Route path="costs" element={<Costs />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
