export interface NotificationDestination {
  id: string;
  provider: string;
  name: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}
export interface NotificationRoute {
  id: string; name: string; enabled: boolean; priority: number; severities: string[]; categories: string[];
  serverIds: string[]; serverTags: string[]; transitions: string[]; destinationIds: string[]; cooldownSeconds: number;
  createdAt: string; updatedAt: string;
}
export interface NotificationDelivery {
  eventId: string; destinationId: string; destinationName: string; eventType: string; findingId: string; findingTitle: string;
  serverId: string; serverName: string; severity: string; category: string; status: string; lastError?: string; attempts: number;
  createdAt: string; lastAttemptAt?: string; deliveredAt?: string; nextAttemptAt?: string;
}
