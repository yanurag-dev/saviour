export interface CPUMetrics {
  usage_percent: number;
  per_core_percent: number[];
  load_avg_1: number;
  load_avg_5: number;
  load_avg_15: number;
}

export interface MemoryMetrics {
  total: number;
  available: number;
  used: number;
  used_percent: number;
  swap_total: number;
  swap_used: number;
  swap_percent: number;
}

export interface DiskMetrics {
  mount_point: string;
  device: string;
  fs_type: string;
  total: number;
  used: number;
  free: number;
  used_percent: number;
  inodes_total: number;
  inodes_used: number;
  inodes_free: number;
}

export interface NetworkMetrics {
  bytes_sent: number;
  bytes_recv: number;
  packets_sent: number;
  packets_recv: number;
  errors_in: number;
  errors_out: number;
  drops_in: number;
  drops_out: number;
}

export interface SystemInfo {
  hostname: string;
  os: string;
  platform: string;
  platform_version: string;
  kernel_version: string;
  uptime: number;
}

export interface SystemMetrics {
  timestamp: string;
  agent_name: string;
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  disk: DiskMetrics[];
  network: NetworkMetrics;
  system_info: SystemInfo;
}

export interface ContainerState {
  id: string;
  name: string;
  image: string;
  image_id: string;
  labels?: Record<string, string>;
  state: string;
  status: string;
  health: string;
  exit_code: number;
  oom_killed: boolean;
  restart_count: number;
  created: string;
  started_at: string;
  finished_at?: string;
  cpu_percent: number;
  memory_usage: number;
  memory_limit: number;
  memory_percent: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  pids: number;
  previous_state?: string;
  last_state_change?: string;
}

export interface Alert {
  id: string;
  agent_name: string;
  alert_type: string;
  severity: 'info' | 'warning' | 'critical';
  message: string;
  details: Record<string, unknown>;
  triggered_at: string;
  resolved_at?: string;
  status: string;
  notified_at?: string;
}

export interface ServerState {
  agent_name: string;
  ec2_instance_id: string;
  status: 'online' | 'offline';
  last_seen: string;
  system_metrics: SystemMetrics;
  containers: ContainerState[];
  active_alerts: Alert[];
}

export interface SSEUpdate {
  agents: ServerState[];
  alerts: Alert[];
  timestamp: number;
}
