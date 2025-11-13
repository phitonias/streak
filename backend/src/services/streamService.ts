import { query, redisClient } from '../config/database';
import { Stream } from '../types';
import { AppError } from '../middleware/errorHandler';
import { logger } from '../utils/logger';

export interface CreateStreamData {
  athleteId: string;
  title: string;
  description?: string;
  sportCategory: string;
  tags?: string[];
}

export interface UpdateStreamData {
  title?: string;
  description?: string;
  sportCategory?: string;
  tags?: string[];
  thumbnailUrl?: string;
}

export class StreamService {
  async createStream(data: CreateStreamData): Promise<any> {
    const { athleteId, title, description, sportCategory, tags } = data;

    // Check if athlete already has an active stream
    const activeStream = await query(
      'SELECT id FROM streams WHERE athlete_id = $1 AND is_live = TRUE',
      [athleteId]
    );

    if (activeStream.rows.length > 0) {
      throw new AppError('You already have an active stream', 400);
    }

    const result = await query(
      `INSERT INTO streams (athlete_id, title, description, sport_category, tags)
       VALUES ($1, $2, $3, $4, $5)
       RETURNING *`,
      [athleteId, title, description || null, sportCategory, tags || []]
    );

    logger.info(`Stream created: ${result.rows[0].id} by athlete ${athleteId}`);

    return result.rows[0];
  }

  async getStreamById(streamId: string): Promise<any> {
    const result = await query(
      `SELECT s.*, u.username, u.display_name, u.avatar_url,
              ap.follower_count, ap.sport_categories
       FROM streams s
       JOIN users u ON s.athlete_id = u.id
       JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE s.id = $1`,
      [streamId]
    );

    if (result.rows.length === 0) {
      throw new AppError('Stream not found', 404);
    }

    const stream = result.rows[0];

    // Get viewer count from Redis if live
    if (stream.is_live) {
      const viewerCount = await this.getViewerCount(streamId);
      stream.current_viewer_count = viewerCount;
    }

    return stream;
  }

  async updateStream(streamId: string, athleteId: string, data: UpdateStreamData): Promise<any> {
    // Verify ownership
    const streamResult = await query(
      'SELECT athlete_id FROM streams WHERE id = $1',
      [streamId]
    );

    if (streamResult.rows.length === 0) {
      throw new AppError('Stream not found', 404);
    }

    if (streamResult.rows[0].athlete_id !== athleteId) {
      throw new AppError('Not authorized to update this stream', 403);
    }

    const updates: string[] = [];
    const values: any[] = [];
    let paramCounter = 1;

    if (data.title !== undefined) {
      updates.push(`title = $${paramCounter++}`);
      values.push(data.title);
    }

    if (data.description !== undefined) {
      updates.push(`description = $${paramCounter++}`);
      values.push(data.description);
    }

    if (data.sportCategory !== undefined) {
      updates.push(`sport_category = $${paramCounter++}`);
      values.push(data.sportCategory);
    }

    if (data.tags !== undefined) {
      updates.push(`tags = $${paramCounter++}`);
      values.push(data.tags);
    }

    if (data.thumbnailUrl !== undefined) {
      updates.push(`thumbnail_url = $${paramCounter++}`);
      values.push(data.thumbnailUrl);
    }

    if (updates.length > 0) {
      values.push(streamId);
      await query(
        `UPDATE streams SET ${updates.join(', ')} WHERE id = $${paramCounter}`,
        values
      );
    }

    logger.info(`Stream updated: ${streamId}`);

    return this.getStreamById(streamId);
  }

  async startStream(athleteId: string, streamId: string): Promise<any> {
    // Verify ownership
    const streamResult = await query(
      'SELECT is_live FROM streams WHERE id = $1 AND athlete_id = $2',
      [streamId, athleteId]
    );

    if (streamResult.rows.length === 0) {
      throw new AppError('Stream not found or not authorized', 404);
    }

    if (streamResult.rows[0].is_live) {
      throw new AppError('Stream is already live', 400);
    }

    // Update stream status
    await query(
      `UPDATE streams
       SET is_live = TRUE, started_at = CURRENT_TIMESTAMP
       WHERE id = $1`,
      [streamId]
    );

    // Update athlete profile
    await query(
      'UPDATE athlete_profiles SET is_live = TRUE WHERE user_id = $1',
      [athleteId]
    );

    // Create stream session
    await query(
      'INSERT INTO stream_sessions (stream_id) VALUES ($1)',
      [streamId]
    );

    // Set stream status in Redis
    await redisClient.set(
      `stream:${streamId}:status`,
      JSON.stringify({ is_live: true, started_at: new Date() }),
      { EX: 86400 } // 24 hours
    );

    logger.info(`Stream started: ${streamId} by athlete ${athleteId}`);

    return this.getStreamById(streamId);
  }

  async stopStream(athleteId: string, streamId: string): Promise<void> {
    // Verify ownership
    const streamResult = await query(
      'SELECT is_live FROM streams WHERE id = $1 AND athlete_id = $2',
      [streamId, athleteId]
    );

    if (streamResult.rows.length === 0) {
      throw new AppError('Stream not found or not authorized', 404);
    }

    if (!streamResult.rows[0].is_live) {
      throw new AppError('Stream is not live', 400);
    }

    // Get current viewer count for peak tracking
    const viewerCount = await this.getViewerCount(streamId);

    // Update stream status
    await query(
      `UPDATE streams
       SET is_live = FALSE, ended_at = CURRENT_TIMESTAMP,
           peak_viewer_count = GREATEST(peak_viewer_count, $2)
       WHERE id = $1`,
      [streamId, viewerCount]
    );

    // Update athlete profile (set is_live to false only if no other streams are live)
    await query(
      `UPDATE athlete_profiles
       SET is_live = EXISTS(
         SELECT 1 FROM streams WHERE athlete_id = $1 AND is_live = TRUE
       )
       WHERE user_id = $1`,
      [athleteId]
    );

    // Close stream session
    await query(
      `UPDATE stream_sessions
       SET ended_at = CURRENT_TIMESTAMP,
           duration_seconds = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - started_at))::INTEGER,
           peak_viewers = $2
       WHERE stream_id = $1 AND ended_at IS NULL`,
      [streamId, viewerCount]
    );

    // Clear Redis data
    await redisClient.del(`stream:${streamId}:status`);
    await redisClient.del(`stream:${streamId}:viewers`);

    logger.info(`Stream stopped: ${streamId}`);
  }

  async getLiveStreams(page: number = 1, limit: number = 20, category?: string): Promise<any> {
    const offset = (page - 1) * limit;
    let queryText = `
      SELECT s.id, s.title, s.sport_category, s.tags, s.thumbnail_url, s.started_at,
             u.id as athlete_id, u.username, u.display_name, u.avatar_url,
             ap.follower_count
      FROM streams s
      JOIN users u ON s.athlete_id = u.id
      JOIN athlete_profiles ap ON u.id = ap.user_id
      WHERE s.is_live = TRUE
    `;

    const params: any[] = [];
    let paramCounter = 1;

    if (category) {
      queryText += ` AND s.sport_category = $${paramCounter++}`;
      params.push(category);
    }

    queryText += ` ORDER BY s.current_viewer_count DESC, s.started_at DESC
                   LIMIT $${paramCounter++} OFFSET $${paramCounter}`;
    params.push(limit, offset);

    const result = await query(queryText, params);

    // Enhance with real-time viewer counts from Redis
    const streams = await Promise.all(
      result.rows.map(async (stream) => {
        const viewerCount = await this.getViewerCount(stream.id);
        return { ...stream, current_viewer_count: viewerCount };
      })
    );

    const countQuery = category
      ? 'SELECT COUNT(*) FROM streams WHERE is_live = TRUE AND sport_category = $1'
      : 'SELECT COUNT(*) FROM streams WHERE is_live = TRUE';

    const countResult = await query(countQuery, category ? [category] : []);
    const total = parseInt(countResult.rows[0].count);

    return {
      streams,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    };
  }

  async getStreamsByAthlete(athleteId: string, page: number = 1, limit: number = 20): Promise<any> {
    const offset = (page - 1) * limit;

    const result = await query(
      `SELECT s.*, u.username, u.display_name, u.avatar_url
       FROM streams s
       JOIN users u ON s.athlete_id = u.id
       WHERE s.athlete_id = $1
       ORDER BY s.created_at DESC
       LIMIT $2 OFFSET $3`,
      [athleteId, limit, offset]
    );

    const countResult = await query(
      'SELECT COUNT(*) FROM streams WHERE athlete_id = $1',
      [athleteId]
    );

    const total = parseInt(countResult.rows[0].count);

    return {
      streams: result.rows,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    };
  }

  async getFollowingStreams(userId: string): Promise<any[]> {
    const result = await query(
      `SELECT s.*, u.username, u.display_name, u.avatar_url, ap.follower_count
       FROM streams s
       JOIN follows f ON s.athlete_id = f.athlete_id
       JOIN users u ON s.athlete_id = u.id
       JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE f.follower_id = $1 AND s.is_live = TRUE
       ORDER BY s.current_viewer_count DESC`,
      [userId]
    );

    // Enhance with real-time viewer counts
    const streams = await Promise.all(
      result.rows.map(async (stream) => {
        const viewerCount = await this.getViewerCount(stream.id);
        return { ...stream, current_viewer_count: viewerCount };
      })
    );

    return streams;
  }

  async addViewer(streamId: string, userId: string): Promise<number> {
    await redisClient.sAdd(`stream:${streamId}:viewers`, userId);
    const count = await redisClient.sCard(`stream:${streamId}:viewers`);

    // Update current viewer count in database periodically (every 10 viewers for efficiency)
    if (count % 10 === 0) {
      await query(
        'UPDATE streams SET current_viewer_count = $1 WHERE id = $2',
        [count, streamId]
      );
    }

    return count;
  }

  async removeViewer(streamId: string, userId: string): Promise<number> {
    await redisClient.sRem(`stream:${streamId}:viewers`, userId);
    const count = await redisClient.sCard(`stream:${streamId}:viewers`);
    return count;
  }

  async getViewerCount(streamId: string): Promise<number> {
    const count = await redisClient.sCard(`stream:${streamId}:viewers`);
    return count || 0;
  }
}

export const streamService = new StreamService();
