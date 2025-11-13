import bcrypt from 'bcrypt';
import jwt from 'jsonwebtoken';
import { v4 as uuidv4 } from 'uuid';
import { query } from '../config/database';
import { config } from '../config';
import { User, UserRole, JWTPayload } from '../types';
import { AppError } from '../middleware/errorHandler';
import { logger } from '../utils/logger';

const SALT_ROUNDS = 10;

export interface RegisterData {
  username: string;
  email: string;
  password: string;
  role: UserRole;
  displayName?: string;
  sportCategories?: string[];
}

export interface LoginData {
  username: string;
  password: string;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  user: Omit<User, 'password_hash'>;
}

export class AuthService {
  async register(data: RegisterData): Promise<AuthTokens> {
    const { username, email, password, role, displayName, sportCategories } = data;

    // Check if user already exists
    const existingUser = await query(
      'SELECT id FROM users WHERE username = $1 OR email = $2',
      [username, email]
    );

    if (existingUser.rows.length > 0) {
      throw new AppError('Username or email already exists', 409);
    }

    // Hash password
    const passwordHash = await bcrypt.hash(password, SALT_ROUNDS);

    // Create user
    const userResult = await query(
      `INSERT INTO users (username, email, password_hash, role, display_name)
       VALUES ($1, $2, $3, $4, $5)
       RETURNING id, username, email, role, display_name, bio, avatar_url, banner_url,
                 created_at, updated_at, is_verified, is_banned`,
      [username, email, passwordHash, role, displayName || username]
    );

    const user = userResult.rows[0];

    // If athlete, create athlete profile
    if (role === UserRole.ATHLETE) {
      const streamKey = this.generateStreamKey();
      await query(
        `INSERT INTO athlete_profiles (user_id, sport_categories, stream_key, rtmp_url)
         VALUES ($1, $2, $3, $4)`,
        [user.id, sportCategories || [], streamKey, `${config.rtmp.serverUrl}/${streamKey}`]
      );
    }

    // Generate tokens
    const tokens = this.generateTokens({
      userId: user.id,
      username: user.username,
      role: user.role,
    });

    logger.info(`User registered: ${username} (${role})`);

    return {
      ...tokens,
      user,
    };
  }

  async login(data: LoginData): Promise<AuthTokens> {
    const { username, password } = data;

    // Find user
    const userResult = await query(
      `SELECT id, username, email, password_hash, role, display_name, bio, avatar_url,
              banner_url, created_at, updated_at, is_verified, is_banned
       FROM users WHERE username = $1`,
      [username]
    );

    if (userResult.rows.length === 0) {
      throw new AppError('Invalid username or password', 401);
    }

    const user = userResult.rows[0];

    // Check if banned
    if (user.is_banned) {
      throw new AppError('Account has been banned', 403);
    }

    // Verify password
    const isPasswordValid = await bcrypt.compare(password, user.password_hash);
    if (!isPasswordValid) {
      throw new AppError('Invalid username or password', 401);
    }

    // Generate tokens
    const tokens = this.generateTokens({
      userId: user.id,
      username: user.username,
      role: user.role,
    });

    // Remove password hash from user object
    const { password_hash, ...userWithoutPassword } = user;

    logger.info(`User logged in: ${username}`);

    return {
      ...tokens,
      user: userWithoutPassword,
    };
  }

  async refreshToken(refreshToken: string): Promise<{ accessToken: string }> {
    try {
      const decoded = jwt.verify(refreshToken, config.jwt.refreshSecret) as JWTPayload;

      // Verify user still exists
      const userResult = await query(
        'SELECT id, username, role, is_banned FROM users WHERE id = $1',
        [decoded.userId]
      );

      if (userResult.rows.length === 0 || userResult.rows[0].is_banned) {
        throw new AppError('Invalid refresh token', 401);
      }

      const user = userResult.rows[0];

      // Generate new access token
      const accessToken = jwt.sign(
        {
          userId: user.id,
          username: user.username,
          role: user.role,
        },
        config.jwt.secret,
        { expiresIn: config.jwt.expiry }
      );

      return { accessToken };
    } catch (error) {
      throw new AppError('Invalid refresh token', 401);
    }
  }

  async getMe(userId: string): Promise<any> {
    const userResult = await query(
      `SELECT u.id, u.username, u.email, u.role, u.display_name, u.bio, u.avatar_url,
              u.banner_url, u.created_at, u.updated_at, u.is_verified,
              ap.sport_categories, ap.stream_key, ap.rtmp_url, ap.follower_count,
              ap.total_view_count, ap.is_live
       FROM users u
       LEFT JOIN athlete_profiles ap ON u.id = ap.user_id
       WHERE u.id = $1`,
      [userId]
    );

    if (userResult.rows.length === 0) {
      throw new AppError('User not found', 404);
    }

    return userResult.rows[0];
  }

  private generateTokens(payload: JWTPayload): { accessToken: string; refreshToken: string } {
    const accessToken = jwt.sign(payload, config.jwt.secret, {
      expiresIn: config.jwt.expiry,
    });

    const refreshToken = jwt.sign(payload, config.jwt.refreshSecret, {
      expiresIn: config.jwt.refreshExpiry,
    });

    return { accessToken, refreshToken };
  }

  private generateStreamKey(): string {
    return `${uuidv4().replace(/-/g, '')}_${Date.now()}`;
  }
}

export const authService = new AuthService();
