import React, { useState, useEffect } from 'react';
import { ServerState } from '../types/api';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import './Charts.css';

interface ChartsProps {
  agents: ServerState[];
}

interface MetricHistory {
  timestamp: number;
  [key: string]: number;
}

export const Charts: React.FC<ChartsProps> = ({ agents }) => {
  const [cpuHistory, setCpuHistory] = useState<MetricHistory[]>([]);
  const [memoryHistory, setMemoryHistory] = useState<MetricHistory[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string>('all');

  useEffect(() => {
    const timestamp = Date.now();

    const newCpuPoint: MetricHistory = { timestamp };
    const newMemoryPoint: MetricHistory = { timestamp };

    if (selectedAgent === 'all') {
      agents.forEach(agent => {
        newCpuPoint[agent.agent_name] = agent.system_metrics.cpu.usage_percent;
        newMemoryPoint[agent.agent_name] = agent.system_metrics.memory.used_percent;
      });
    } else {
      const agent = agents.find(a => a.agent_name === selectedAgent);
      if (agent) {
        newCpuPoint[agent.agent_name] = agent.system_metrics.cpu.usage_percent;
        newMemoryPoint[agent.agent_name] = agent.system_metrics.memory.used_percent;
      }
    }

    setCpuHistory(prev => [...prev.slice(-19), newCpuPoint]);
    setMemoryHistory(prev => [...prev.slice(-19), newMemoryPoint]);
  }, [agents, selectedAgent]);

  const formatTime = (timestamp: number) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });
  };

  const colors = [
    'var(--accent-primary)',
    'var(--status-success)',
    'var(--status-info)',
    'var(--status-warning)',
    '#00ff88',
    '#ffaa33',
  ];

  const agentNames = selectedAgent === 'all'
    ? agents.map(a => a.agent_name)
    : [selectedAgent];

  return (
    <div className="charts-page">
      <div className="page-header">
        <h2>Real-time Metrics</h2>
        <p>Live system performance charts</p>
      </div>

      <div className="charts-controls">
        <label className="chart-label">
          Agent:
          <select
            className="chart-select"
            value={selectedAgent}
            onChange={(e) => setSelectedAgent(e.target.value)}
          >
            <option value="all">All Agents</option>
            {agents.map(agent => (
              <option key={agent.agent_name} value={agent.agent_name}>
                {agent.agent_name}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="charts-grid">
        <div className="chart-container">
          <h3 className="chart-title">CPU Usage (%)</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={cpuHistory}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatTime}
                stroke="var(--text-muted)"
                style={{ fontSize: '0.75rem', fontFamily: 'var(--font-mono)' }}
              />
              <YAxis
                domain={[0, 100]}
                stroke="var(--text-muted)"
                style={{ fontSize: '0.75rem', fontFamily: 'var(--font-mono)' }}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 0,
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.8rem',
                }}
                labelFormatter={formatTime}
              />
              <Legend
                wrapperStyle={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.75rem',
                }}
              />
              {agentNames.map((name, index) => (
                <Line
                  key={name}
                  type="monotone"
                  dataKey={name}
                  stroke={colors[index % colors.length]}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-container">
          <h3 className="chart-title">Memory Usage (%)</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={memoryHistory}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatTime}
                stroke="var(--text-muted)"
                style={{ fontSize: '0.75rem', fontFamily: 'var(--font-mono)' }}
              />
              <YAxis
                domain={[0, 100]}
                stroke="var(--text-muted)"
                style={{ fontSize: '0.75rem', fontFamily: 'var(--font-mono)' }}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 0,
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.8rem',
                }}
                labelFormatter={formatTime}
              />
              <Legend
                wrapperStyle={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.75rem',
                }}
              />
              {agentNames.map((name, index) => (
                <Line
                  key={name}
                  type="monotone"
                  dataKey={name}
                  stroke={colors[index % colors.length]}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
};
