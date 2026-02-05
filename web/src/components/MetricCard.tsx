import React from 'react';
import './MetricCard.css';

interface MetricCardProps {
  label: string;
  value: string;
  status?: 'success' | 'warning' | 'error' | 'neutral';
  sublabel?: string;
  trend?: 'up' | 'down' | 'stable';
}

export const MetricCard: React.FC<MetricCardProps> = ({
  label,
  value,
  status = 'neutral',
  sublabel,
  trend
}) => {
  return (
    <div className={`metric-card metric-card--${status}`}>
      <div className="metric-card__label">{label}</div>
      <div className="metric-card__value">
        {value}
        {trend && <span className={`metric-card__trend metric-card__trend--${trend}`} />}
      </div>
      {sublabel && <div className="metric-card__sublabel">{sublabel}</div>}
    </div>
  );
};
