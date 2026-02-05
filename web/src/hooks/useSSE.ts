import { useEffect, useState, useRef } from 'react';
import { SSEUpdate } from '../types/api';
import { API_ENDPOINTS } from '../lib/config';

export function useSSE() {
  const [data, setData] = useState<SSEUpdate | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const connect = () => {
      try {
        const eventSource = new EventSource(API_ENDPOINTS.EVENTS);
        eventSourceRef.current = eventSource;

        eventSource.onopen = () => {
          setIsConnected(true);
          setError(null);
        };

        eventSource.onmessage = (event) => {
          try {
            const parsed = JSON.parse(event.data) as SSEUpdate;
            setData(parsed);
          } catch (err) {
            console.error('Failed to parse SSE data:', err);
          }
        };

        eventSource.onerror = (err) => {
          setIsConnected(false);
          const errorMsg = eventSource.readyState === EventSource.CLOSED 
            ? 'SSE connection closed. Check CORS and server configuration.'
            : 'SSE connection error';
          setError(new Error(errorMsg));
          
          // Only reconnect if connection was established before
          if (eventSource.readyState === EventSource.CLOSED) {
            eventSource.close();
            // Reconnect after 5 seconds
            setTimeout(connect, 5000);
          }
        };
      } catch (err) {
        setError(err as Error);
      }
    };

    connect();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  return { data, error, isConnected };
}
