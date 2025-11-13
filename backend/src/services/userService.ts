import { query } from '../config/database';
import { User } from '../types';
import { AppError } from '../middleware/errorHandler';
import { logger } from '../utils/logger';

export interface UpdateProfileData {
  displayName?: string;
  bio?: string;
  avatarUrl?: string;
  bannerUrl?: string;
  sportCategories?: string[];
}

export class UserService {
  async getUserById(userId: string): Promise<any> {
    const result = await query(
      `SELECT u.id, u.username, u.role, u.display_name, u.bio, u.avatar_url, u.banner_url,
              u.created_at, u.is_verified,
              ap.sport_categories, ap.follower_count, ap.total_view_count, ap.is_live
       FROM users u
       LEFT JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE u.id = $1 AND u.is_banned = FALSE`,
      [userId]
    );

    if (result.rows.length === 0) {
      throw new AppError('User not found', 404);
    }

    return result.rows[0];
  }

  async getUserByUsername(username: string): Promise<any> {
    const result = await query(
      `SELECT u.id, u.username, u.role, u.display_name, u.bio, u.avatar_url, u.banner_url,
              u.created_at, u.is_verified,
              ap.sport_categories, ap.follower_count, ap.total_view_count, ap.is_live
       FROM users u
       LEFT JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE u.username = $1 AND u.is_banned = FALSE`,
      [username]
    );

    if (result.rows.length === 0) {
      throw new AppError('User not found', 404);
    }

    return result.rows[0];
  }

  async updateProfile(userId: string, data: UpdateProfileData): Promise<any> {
    const updates: string[] = [];
    const values: any[] = [];
    let paramCounter = 1;

    if (data.displayName !== undefined) {
      updates.push(`display_name = $${paramCounter++}`);
      values.push(data.displayName);
    }

    if (data.bio !== undefined) {
      updates.push(`bio = $${paramCounter++}`);
      values.push(data.bio);
    }

    if (data.avatarUrl !== undefined) {
      updates.push(`avatar_url = $${paramCounter++}`);
      values.push(data.avatarUrl);
    }

    if (data.bannerUrl !== undefined) {
      updates.push(`banner_url = $${paramCounter++}`);
      values.push(data.bannerUrl);
    }

    if (updates.length > 0) {
      values.push(userId);
      await query(
        `UPDATE users SET ${updates.join(', ')}, updated_at = CURRENT_TIMESTAMP
         WHERE id = $${paramCounter}`,
        values
      );
    }

    // Update athlete profile if sport categories provided
    if (data.sportCategories !== undefined) {
      await query(
        'UPDATE athlete_profiles SET sport_categories = $1 WHERE user_id = $2',
        [data.sportCategories, userId]
      );
    }

    logger.info(`Profile updated for user: ${userId}`);

    return this.getUserById(userId);
  }

  async followUser(followerId: string, athleteId: string): Promise<void> {
    // Check if athlete exists and is an athlete
    const athleteResult = await query(
      'SELECT role FROM users WHERE id = $1',
      [athleteId]
    );

    if (athleteResult.rows.length === 0) {
      throw new AppError('Athlete not found', 404);
    }

    if (athleteResult.rows[0].role !== 'athlete') {
      throw new AppError('Can only follow athletes', 400);
    }

    if (followerId === athleteId) {
      throw new AppError('Cannot follow yourself', 400);
    }

    // Check if already following
    const existingFollow = await query(
      'SELECT 1 FROM follows WHERE follower_id = $1 AND athlete_id = $2',
      [followerId, athleteId]
    );

    if (existingFollow.rows.length > 0) {
      throw new AppError('Already following this athlete', 400);
    }

    // Create follow relationship
    await query(
      'INSERT INTO follows (follower_id, athlete_id) VALUES ($1, $2)',
      [followerId, athleteId]
    );

    logger.info(`User ${followerId} followed athlete ${athleteId}`);
  }

  async unfollowUser(followerId: string, athleteId: string): Promise<void> {
    const result = await query(
      'DELETE FROM follows WHERE follower_id = $1 AND athlete_id = $2',
      [followerId, athleteId]
    );

    if (result.rowCount === 0) {
      throw new AppError('Not following this athlete', 400);
    }

    logger.info(`User ${followerId} unfollowed athlete ${athleteId}`);
  }

  async getFollowers(athleteId: string, page: number = 1, limit: number = 20): Promise<any> {
    const offset = (page - 1) * limit;

    const result = await query(
      `SELECT u.id, u.username, u.display_name, u.avatar_url, f.created_at as followed_at
       FROM follows f
       JOIN users u ON f.follower_id = u.id
       WHERE f.athlete_id = $1 AND u.is_banned = FALSE
       ORDER BY f.created_at DESC
       LIMIT $2 OFFSET $3`,
      [athleteId, limit, offset]
    );

    const countResult = await query(
      'SELECT COUNT(*) FROM follows WHERE athlete_id = $1',
      [athleteId]
    );

    const total = parseInt(countResult.rows[0].count);

    return {
      followers: result.rows,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    };
  }

  async getFollowing(userId: string, page: number = 1, limit: number = 20): Promise<any> {
    const offset = (page - 1) * limit;

    const result = await query(
      `SELECT u.id, u.username, u.display_name, u.avatar_url,
              ap.is_live, f.created_at as followed_at
       FROM follows f
       JOIN users u ON f.athlete_id = u.id
       JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE f.follower_id = $1 AND u.is_banned = FALSE
       ORDER BY ap.is_live DESC, f.created_at DESC
       LIMIT $2 OFFSET $3`,
      [userId, limit, offset]
    );

    const countResult = await query(
      'SELECT COUNT(*) FROM follows WHERE follower_id = $1',
      [userId]
    );

    const total = parseInt(countResult.rows[0].count);

    return {
      following: result.rows,
      pagination: {
        page,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    };
  }

  async isFollowing(followerId: string, athleteId: string): Promise<boolean> {
    const result = await query(
      'SELECT 1 FROM follows WHERE follower_id = $1 AND athlete_id = $2',
      [followerId, athleteId]
    );

    return result.rows.length > 0;
  }
}

export const userService = new UserService();
