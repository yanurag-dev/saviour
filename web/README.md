# Saviour Dashboard

Production-grade web interface for the Saviour monitoring platform.

## Tech Stack

- **React 18** + **TypeScript** - Type-safe component architecture
- **Vite** - Lightning-fast build tool and dev server
- **Recharts** - Composable charting library for real-time metrics
- **Server-Sent Events (SSE)** - Real-time data streaming from backend

## Design Philosophy

**Industrial Terminal Aesthetic**

This dashboard embraces a technical brutalist design inspired by CRT terminals and professional monitoring tools. Key characteristics:

- **Monospace Typography**: JetBrains Mono throughout for data authenticity
- **High Contrast Color System**: Charcoal base (#0f0f0f) with vivid status indicators
- **Amber Accents**: Warm #ff9500 for active states and highlights (CRT-inspired)
- **Dense Information Display**: Efficient use of space while maintaining breathing room
- **Zero Rounded Corners**: Sharp, technical aesthetic
- **Vivid Status Colors**: Emerald green (success), ruby red (error), golden yellow (warning)

## Features

### 1. Agent Overview
- Grid view of all monitoring agents
- Real-time CPU, memory, and disk metrics with animated progress bars
- Online/offline status indicators
- Agent metadata (platform, uptime, container count)

### 2. Container Monitoring
- Comprehensive table of all containers across infrastructure
- Filter by state (all/running/stopped)
- Search by container name, image, or agent
- Per-container resource metrics (CPU, memory, restarts)
- Health status tracking

### 3. Alert Dashboard
- Severity-based alert categorization (critical/warning/info)
- Real-time alert feed with auto-sorting by timestamp
- Detailed alert metadata and context
- Visual severity indicators

### 4. Real-time Charts
- Live CPU and memory usage graphs
- Per-agent or aggregate view selection
- 20-point rolling history
- Synchronized updates via SSE

## Setup

### Prerequisites

- Node.js 18+ or pnpm/yarn
- Running Saviour backend server (default: http://localhost:8080)

### Installation

```bash
cd web
npm install
```

### Development

```bash
npm run dev
```

Dashboard will be available at `http://localhost:3000` with hot module replacement.

### Build for Production

```bash
npm run build
```

Optimized production build will be in `dist/` directory.

### Preview Production Build

```bash
npm run preview
```

## Configuration

### API Endpoint

Create a `.env` file in the `web/` directory:

```env
VITE_API_URL=http://localhost:8080
```

The dashboard uses Vite's proxy in development mode to avoid CORS issues. In production, ensure your backend has appropriate CORS headers configured.

## Architecture

### Real-time Data Flow

```
Backend SSE (/api/v1/events)
         ↓
  useSSE Hook (EventSource)
         ↓
   App State (data)
         ↓
  Page Components (AgentOverview, Containers, Alerts, Charts)
```

### Type Safety

All API responses are strongly typed using TypeScript interfaces in `src/types/api.ts`, matching the Go backend structs exactly. This ensures compile-time safety when consuming backend data.

### Component Structure

```
src/
├── components/        # Reusable UI components
│   ├── MetricCard     # Numeric metric display
│   ├── StatusBadge    # Status indicator with color
│   └── DataTable      # Generic table component
├── pages/             # Main application views
│   ├── AgentOverview  # Agent grid view
│   ├── Containers     # Container monitoring
│   ├── Alerts         # Alert dashboard
│   └── Charts         # Real-time graphs
├── hooks/             # Custom React hooks
│   └── useSSE         # Server-Sent Events connection
├── lib/               # Utilities
│   ├── config.ts      # API endpoints
│   └── utils.ts       # Formatting helpers
└── types/             # TypeScript definitions
    └── api.ts         # Backend API types
```

## Performance

- **Bundle Size**: ~200KB gzipped (including React + Recharts)
- **Initial Load**: <500ms on broadband
- **SSE Overhead**: ~1-2KB per update (every 5 seconds)
- **Chart Updates**: Zero-copy data updates, no re-renders on non-visible pages

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

Requires native EventSource (SSE) support.

## Customization

### Theming

All design tokens are defined as CSS variables in `src/App.css`:

```css
:root {
  --bg-primary: #0f0f0f;
  --accent-primary: #ff9500;
  --status-success: #00ff88;
  /* ... */
}
```

### Adding New Pages

1. Create component in `src/pages/`
2. Add route in `App.tsx`
3. Add navigation link in sidebar
4. Consume SSE data via props

## License

MIT
