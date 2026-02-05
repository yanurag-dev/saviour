import React from 'react';
import { Alert } from '../types/api';
import { StatusBadge } from '../components/StatusBadge';
import { formatTimestamp } from '../lib/utils';
import './Alerts.css';

interface AlertsProps {
  alerts: Alert[];
}

export const Alerts: React.FC<AlertsProps> = ({ alerts }) => {
  const sortedAlerts = [...alerts].sort((a, b) =>
    new Date(b.triggered_at).getTime() - new Date(a.triggered_at).getTime()
  );

  const criticalCount = alerts.filter(a => a.severity === 'critical').length;
  const warningCount = alerts.filter(a => a.severity === 'warning').length;

  return (
    <div className="alerts-page">
      <div className="page-header">
        <h2>Alert Dashboard</h2>
        <p>Active alerts and incident history</p>
      </div>

      <div className="alerts-summary">
        <div className="alert-stat alert-stat--critical">
          <div className="alert-stat__value">{criticalCount}</div>
          <div className="alert-stat__label">Critical</div>
        </div>
        <div className="alert-stat alert-stat--warning">
          <div className="alert-stat__value">{warningCount}</div>
          <div className="alert-stat__label">Warning</div>
        </div>
        <div className="alert-stat">
          <div className="alert-stat__value">{alerts.length - criticalCount - warningCount}</div>
          <div className="alert-stat__label">Info</div>
        </div>
      </div>

      <div className="alerts-list">
        {sortedAlerts.length === 0 ? (
          <div className="alerts-empty">
            <div className="alerts-empty__icon">✓</div>
            <div className="alerts-empty__text">No active alerts</div>
            <div className="alerts-empty__subtext">All systems operational</div>
          </div>
        ) : (
          sortedAlerts.map((alert) => (
            <div key={alert.id} className={`alert-card alert-card--${alert.severity}`}>
              <div className="alert-card__header">
                <StatusBadge status={alert.severity} />
                <span className="alert-card__time">{formatTimestamp(alert.triggered_at)}</span>
              </div>

              <div className="alert-card__body">
                <div className="alert-card__type">{alert.alert_type}</div>
                <div className="alert-card__message">{alert.message}</div>
                <div className="alert-card__agent">Agent: {alert.agent_name}</div>
              </div>

              {Object.keys(alert.details).length > 0 && (
                <div className="alert-card__details">
                  {Object.entries(alert.details).map(([key, value]) => (
                    <div key={key} className="alert-detail">
                      <span className="alert-detail__key">{key}:</span>
                      <span className="alert-detail__value">{String(value)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
};
