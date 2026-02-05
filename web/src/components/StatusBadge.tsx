import React from 'react';
import './StatusBadge.css';

interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, size = 'md' }) => {
  const normalizedStatus = status.toLowerCase();

  return (
    <span className={`status-badge status-badge--${size} status-badge--${normalizedStatus}`}>
      <span className="status-badge__dot" />
      {status}
    </span>
  );
};
