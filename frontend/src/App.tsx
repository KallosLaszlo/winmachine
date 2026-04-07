import { useState } from 'react';
import './App.css';
import Dashboard from './pages/Dashboard';
import Settings from './pages/Settings';
import TimeBrowser from './pages/TimeBrowser';

type Page = 'dashboard' | 'timebrowser' | 'settings';

function App() {
  const [page, setPage] = useState<Page>('dashboard');

  return (
    <div className="app-layout">
      <nav className="sidebar">
        <div className="sidebar-brand">
          <div className="brand-icon">⏱</div>
          <span className="brand-name">WinMachine</span>
        </div>
        <div className="nav-items">
          <button
            className={`nav-item ${page === 'dashboard' ? 'active' : ''}`}
            onClick={() => setPage('dashboard')}
          >
            <span className="nav-icon">📊</span>
            Dashboard
          </button>
          <button
            className={`nav-item ${page === 'timebrowser' ? 'active' : ''}`}
            onClick={() => setPage('timebrowser')}
          >
            <span className="nav-icon">🕐</span>
            Time Browser
          </button>
          <button
            className={`nav-item ${page === 'settings' ? 'active' : ''}`}
            onClick={() => setPage('settings')}
          >
            <span className="nav-icon">⚙️</span>
            Settings
          </button>
        </div>
        <div className="sidebar-footer">
          <span className="version">v0.1.9</span>
        </div>
      </nav>
      <main className="content">
        {page === 'dashboard' && <Dashboard />}
        {page === 'timebrowser' && <TimeBrowser />}
        {page === 'settings' && <Settings />}
      </main>
    </div>
  );
}

export default App;
