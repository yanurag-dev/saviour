import { useState } from 'react';
import { useSSE } from './hooks/useSSE';
import { AgentOverview } from './pages/AgentOverview';
import { Containers } from './pages/Containers';
import { Alerts } from './pages/Alerts';
import { Charts } from './pages/Charts';
import './App.css';

type Page = 'overview' | 'containers' | 'alerts' | 'charts';

function App() {
  const [currentPage, setCurrentPage] = useState<Page>('overview');
  const { data, isConnected } = useSSE();

  const agents = data?.agents || [];
  const alerts = data?.alerts || [];

  const renderPage = () => {
    if (!data) {
      return (
        <div className="loading-container">
          <div className="loading-spinner" />
          <p>Connecting to monitoring server...</p>
        </div>
      );
    }

    switch (currentPage) {
      case 'overview':
        return <AgentOverview agents={agents} />;
      case 'containers':
        return <Containers agents={agents} />;
      case 'alerts':
        return <Alerts alerts={alerts} />;
      case 'charts':
        return <Charts agents={agents} />;
      default:
        return <AgentOverview agents={agents} />;
    }
  };

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-logo">
          <h2>SAVIOUR</h2>
        </div>
        <nav className="sidebar-nav">
          <a
            href="#overview"
            className={`nav-link ${currentPage === 'overview' ? 'active' : ''}`}
            onClick={(e) => {
              e.preventDefault();
              setCurrentPage('overview');
            }}
          >
            Overview
          </a>
          <a
            href="#containers"
            className={`nav-link ${currentPage === 'containers' ? 'active' : ''}`}
            onClick={(e) => {
              e.preventDefault();
              setCurrentPage('containers');
            }}
          >
            Containers
          </a>
          <a
            href="#alerts"
            className={`nav-link ${currentPage === 'alerts' ? 'active' : ''}`}
            onClick={(e) => {
              e.preventDefault();
              setCurrentPage('alerts');
            }}
          >
            Alerts
          </a>
          <a
            href="#charts"
            className={`nav-link ${currentPage === 'charts' ? 'active' : ''}`}
            onClick={(e) => {
              e.preventDefault();
              setCurrentPage('charts');
            }}
          >
            Charts
          </a>
        </nav>
      </aside>

      <header className="app-header">
        <div className="header-title">
          <h1>Infrastructure Monitor</h1>
        </div>
        <div className="connection-status">
          <span className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`} />
          <span>{isConnected ? 'LIVE' : 'DISCONNECTED'}</span>
        </div>
      </header>

      <main className="main-content">
        {renderPage()}
      </main>
    </div>
  );
}

export default App;
