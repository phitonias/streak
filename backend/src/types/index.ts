import { Request } from 'express';

// User roles
export enum UserRole {
  ATHLETE = 'athlete',
  FOLLOWER = 'follower',
}

// User interface
export interface User {
  id: string;
  username: string;
  email: string;
  password_hash: string;
  role: UserRole;
  display_name: string | null;
  bio: string | null;
  avatar_url: string | null;
  banner_url: string | null;
  created_at: Date;
  updated_at: Date;
  is_verified: boolean;
  is_banned: boolean;
}

// Athlete profile interface
export interface AthleteProfile {
  user_id: string;
  sport_categories: string[];
  stream_key: string;
  rtmp_url: string | null;
  follower_count: number;
  total_view_count: number;
  is_live: boolean;
  created_at: Date;
}

// Stream interface
export interface Stream {
  id: string;
  athlete_id: string;
  title: string;
  description: string | null;
  sport_category: string;
  tags: string[];
  thumbnail_url: string | null;
  is_live: boolean;
  started_at: Date | null;
  ended_at: Date | null;
  current_viewer_count: number;
  peak_viewer_count: number;
  created_at: Date;
}

// Stream session interface
export interface StreamSession {
  id: string;
  stream_id: string;
  started_at: Date;
  ended_at: Date | null;
  duration_seconds: number | null;
  average_viewers: number;
  peak_viewers: number;
  total_messages: number;
}

// Follow relationship interface
export interface Follow {
  follower_id: string;
  athlete_id: string;
  created_at: Date;
  notifications_enabled: boolean;
}

// Chat message interface
export interface ChatMessage {
  _id?: any;
  stream_id: string;
  user_id: string;
  username: string;
  message: string;
  timestamp: Date;
  is_deleted: boolean;
}

// JWT payload interface
export interface JWTPayload {
  userId: string;
  username: string;
  role: UserRole;
  iat?: number;
  exp?: number;
}

// Authenticated request interface
export interface AuthRequest extends Request {
  user?: JWTPayload;
}

// Stream statistics interface
export interface StreamStats {
  stream_id: string;
  total_sessions: number;
  total_duration_seconds: number;
  average_viewers: number;
  peak_viewers: number;
  total_messages: number;
}

// API response interface
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// Pagination interface
export interface Pagination {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

export interface PaginatedResponse<T = any> extends ApiResponse<T> {
  pagination?: Pagination;
}

// Sport categories
export const SPORT_CATEGORIES = [
  'basketball',
  'football',
  'soccer',
  'baseball',
  'tennis',
  'golf',
  'running',
  'cycling',
  'swimming',
  'hockey',
  'martial-arts',
  'esports',
  'fitness',
  'other',
] as const;

export type SportCategory = typeof SPORT_CATEGORIES[number];

// WebSocket events
export enum SocketEvent {
  // Client to server
  JOIN_STREAM = 'join_stream',
  LEAVE_STREAM = 'leave_stream',
  SEND_MESSAGE = 'send_message',

  // Server to client
  MESSAGE = 'message',
  VIEWER_COUNT_UPDATE = 'viewer_count_update',
  STREAM_STARTED = 'stream_started',
  STREAM_ENDED = 'stream_ended',
  USER_JOINED = 'user_joined',
  USER_LEFT = 'user_left',
  ERROR = 'error',
}
