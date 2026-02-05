import React from 'react';
import { ServerState } from '../types/api';
import { MetricCard } from '../components/MetricCard';
import { StatusBadge } from '../components/StatusBadge';
import { formatUptime, formatPercentage, formatTimestamp } from '../lib/utils';
import './AgentOverview.css';

interface AgentOverviewProps {
  agents: ServerState[];
}

export const AgentOverview: React.FC<AgentOverviewProps> = ({ agents }) => {
  const onlineCount = agents.filter(a => a.status === 'online').length;
  const totalContainers = agents.reduce((sum, a) => sum + (a.containers?.length || 0), 0);
  const runningContainers = agents.reduce(
    (sum, a) => sum + (a.containers?.filter(c => c.state === 'running').length || 0),
    0
  );

  return (
    <div className="agent-overview">
      <div className="page-header">
        <h2>Infrastructure Overview</h2>
        <p>Real-time monitoring of all agents and system metrics</p>
      </div>

      <div className="overview-stats">
        <MetricCard
          label="Agents Online"
          value={`${onlineCount}/${agents.length}`}
          status={onlineCount === agents.length ? 'success' : 'warning'}
        />
        <MetricCard
          label="Containers"
          value={`${runningContainers}/${totalContainers}`}
          sublabel="running"
          status={runningContainers === totalContainers ? 'success' : 'neutral'}
        />
        <MetricCard
          label="Total Alerts"
          value={String(agents.reduce((sum, a) => sum + (a.active_alerts?.length || 0), 0))}
          status="neutral"
        />
      </div>

      <div className="agents-grid">
        {agents.map((agent) => (
          <div key={agent.agent_name} className="agent-card">
            <div className="agent-card__header">
              <div>
                <h3 className="agent-card__name">{agent.agent_name}</h3>
                <div className="agent-card__meta">
                  <StatusBadge status={agent.status} size="sm" />
                  <span className="agent-card__timestamp">
                    {formatTimestamp(agent.last_seen)}
                  </span>
                </div>
              </div>
            </div>

            <div className="agent-card__metrics">
              {agent.system_metrics?.cpu && (
                <div className="metric-row">
                  <span className="metric-row__label">CPU</span>
                  <div className="metric-row__bar">
                    <div
                      className="metric-row__fill"
                      style={{
                        width: `${agent.system_metrics.cpu.usage_percent || 0}%`,
                        background: (agent.system_metrics.cpu.usage_percent || 0) > 80
                          ? 'var(--status-error)'
                          : (agent.system_metrics.cpu.usage_percent || 0) > 60
                          ? 'var(--status-warning)'
                          : 'var(--status-success)'
                      }}
                    />
                  </div>
                  <span className="metric-row__value">
                    {formatPercentage(agent.system_metrics.cpu.usage_percent || 0)}
                  </span>
                </div>
              )}

              {agent.system_metrics?.memory && (
                <div className="metric-row">
                  <span className="metric-row__label">MEM</span>
                  <div className="metric-row__bar">
                    <div
                      className="metric-row__fill"
                      style={{
                        width: `${agent.system_metrics.memory.used_percent || 0}%`,
                        background: (agent.system_metrics.memory.used_percent || 0) > 85
                          ? 'var(--status-error)'
                          : (agent.system_metrics.memory.used_percent || 0) > 70
                          ? 'var(--status-warning)'
                          : 'var(--status-success)'
                      }}
                    />
                  </div>
                  <span className="metric-row__value">
                    {formatPercentage(agent.system_metrics.memory.used_percent || 0)}
                  </span>
                </div>
              )}

              {agent.system_metrics?.disk && agent.system_metrics.disk.length > 0 && (
                <div className="metric-row">
                  <span className="metric-row__label">DISK</span>
                  <div className="metric-row__bar">
                    <div
                      className="metric-row__fill"
                      style={{
                        width: `${agent.system_metrics.disk[0].used_percent}%`,
                        background: agent.system_metrics.disk[0].used_percent > 90
                          ? 'var(--status-error)'
                          : agent.system_metrics.disk[0].used_percent > 75
                          ? 'var(--status-warning)'
                          : 'var(--status-success)'
                      }}
                    />
                  </div>
                  <span className="metric-row__value">
                    {formatPercentage(agent.system_metrics.disk[0].used_percent)}
                  </span>
                </div>
              )}
            </div>

            <div className="agent-card__footer">
              <div className="agent-card__info">
                <span>{agent.system_metrics?.system_info?.platform || 'N/A'}</span>
                <span>•</span>
                <span>↑ {formatUptime(agent.system_metrics?.system_info?.uptime || 0)}</span>
                <span>•</span>
                <span>{agent.containers?.length || 0} containers</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
