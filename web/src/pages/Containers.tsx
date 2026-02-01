import React, { useState } from 'react';
import { ServerState, ContainerState } from '../types/api';
import { DataTable } from '../components/DataTable';
import { StatusBadge } from '../components/StatusBadge';
import { formatBytes, formatPercentage } from '../lib/utils';
import './Containers.css';

interface ContainersProps {
  agents: ServerState[];
}

export const Containers: React.FC<ContainersProps> = ({ agents }) => {
  const [filter, setFilter] = useState<'all' | 'running' | 'stopped'>('all');
  const [searchTerm, setSearchTerm] = useState('');

  const allContainers = agents.flatMap(agent =>
    (agent.containers || []).map(container => ({
      ...container,
      agent_name: agent.agent_name,
    }))
  );

  const filteredContainers = allContainers.filter(container => {
    const matchesFilter =
      filter === 'all' ||
      (filter === 'running' && container.state === 'running') ||
      (filter === 'stopped' && container.state !== 'running');

    const matchesSearch =
      container.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      container.image.toLowerCase().includes(searchTerm.toLowerCase()) ||
      container.agent_name.toLowerCase().includes(searchTerm.toLowerCase());

    return matchesFilter && matchesSearch;
  });

  const columns = [
    {
      key: 'name',
      label: 'Container',
      render: (c: ContainerState & { agent_name: string }) => (
        <div className="container-cell">
          <div className="container-cell__name">{c.name}</div>
          <div className="container-cell__image">{c.image}</div>
        </div>
      ),
    },
    {
      key: 'agent_name',
      label: 'Agent',
      render: (c: ContainerState & { agent_name: string }) => (
        <span className="agent-label">{c.agent_name}</span>
      ),
    },
    {
      key: 'state',
      label: 'State',
      align: 'center' as const,
      render: (c: ContainerState & { agent_name: string }) => (
        <StatusBadge status={c.state} size="sm" />
      ),
    },
    {
      key: 'health',
      label: 'Health',
      align: 'center' as const,
      render: (c: ContainerState & { agent_name: string }) => (
        c.health !== 'none' ? <StatusBadge status={c.health} size="sm" /> : <span>—</span>
      ),
    },
    {
      key: 'cpu',
      label: 'CPU',
      align: 'right' as const,
      render: (c: ContainerState & { agent_name: string }) => (
        <span className={c.cpu_percent > 80 ? 'metric-high' : ''}>
          {formatPercentage(c.cpu_percent)}
        </span>
      ),
    },
    {
      key: 'memory',
      label: 'Memory',
      align: 'right' as const,
      render: (c: ContainerState & { agent_name: string }) => (
        <div className="memory-cell">
          <span className={c.memory_percent > 90 ? 'metric-high' : ''}>
            {formatPercentage(c.memory_percent)}
          </span>
          <span className="memory-cell__usage">
            {formatBytes(c.memory_usage)} / {formatBytes(c.memory_limit)}
          </span>
        </div>
      ),
    },
    {
      key: 'restarts',
      label: 'Restarts',
      align: 'center' as const,
      render: (c: ContainerState & { agent_name: string }) => (
        <span className={c.restart_count > 5 ? 'metric-warning' : ''}>
          {c.restart_count}
        </span>
      ),
    },
  ];

  return (
    <div className="containers-page">
      <div className="page-header">
        <h2>Container Monitoring</h2>
        <p>All containers across infrastructure</p>
      </div>

      <div className="containers-controls">
        <div className="filter-group">
          <button
            className={`filter-btn ${filter === 'all' ? 'filter-btn--active' : ''}`}
            onClick={() => setFilter('all')}
          >
            All ({allContainers.length})
          </button>
          <button
            className={`filter-btn ${filter === 'running' ? 'filter-btn--active' : ''}`}
            onClick={() => setFilter('running')}
          >
            Running ({allContainers.filter(c => c.state === 'running').length})
          </button>
          <button
            className={`filter-btn ${filter === 'stopped' ? 'filter-btn--active' : ''}`}
            onClick={() => setFilter('stopped')}
          >
            Stopped ({allContainers.filter(c => c.state !== 'running').length})
          </button>
        </div>

        <input
          type="text"
          className="search-input"
          placeholder="Search containers..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      <DataTable
        data={filteredContainers}
        columns={columns}
        keyExtractor={(c) => c.id}
      />
    </div>
  );
};
