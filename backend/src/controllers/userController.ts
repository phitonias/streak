import { Request, Response, NextFunction } from 'express';
import { userService } from '../services/userService';
import { AuthRequest } from '../types';
import { AppError } from '../middleware/errorHandler';

export class UserController {
  async getUserById(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { id } = req.params;
      const user = await userService.getUserById(id);

      res.status(200).json({
        success: true,
        data: user,
      });
    } catch (error) {
      next(error);
    }
  }

  async getUserByUsername(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { username } = req.params;
      const user = await userService.getUserByUsername(username);

      res.status(200).json({
        success: true,
        data: user,
      });
    } catch (error) {
      next(error);
    }
  }

  async updateProfile(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { displayName, bio, avatarUrl, bannerUrl, sportCategories } = req.body;

      const updatedUser = await userService.updateProfile(req.user.userId, {
        displayName,
        bio,
        avatarUrl,
        bannerUrl,
        sportCategories,
      });

      res.status(200).json({
        success: true,
        data: updatedUser,
        message: 'Profile updated successfully',
      });
    } catch (error) {
      next(error);
    }
  }

  async followUser(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id: athleteId } = req.params;
      await userService.followUser(req.user.userId, athleteId);

      res.status(200).json({
        success: true,
        message: 'Successfully followed user',
      });
    } catch (error) {
      next(error);
    }
  }

  async unfollowUser(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id: athleteId } = req.params;
      await userService.unfollowUser(req.user.userId, athleteId);

      res.status(200).json({
        success: true,
        message: 'Successfully unfollowed user',
      });
    } catch (error) {
      next(error);
    }
  }

  async getFollowers(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { id } = req.params;
      const page = parseInt(req.query.page as string) || 1;
      const limit = parseInt(req.query.limit as string) || 20;

      const result = await userService.getFollowers(id, page, limit);

      res.status(200).json({
        success: true,
        data: result.followers,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  async getFollowing(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { id } = req.params;
      const page = parseInt(req.query.page as string) || 1;
      const limit = parseInt(req.query.limit as string) || 20;

      const result = await userService.getFollowing(id, page, limit);

      res.status(200).json({
        success: true,
        data: result.following,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  async checkFollowing(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id: athleteId } = req.params;
      const isFollowing = await userService.isFollowing(req.user.userId, athleteId);

      res.status(200).json({
        success: true,
        data: { isFollowing },
      });
    } catch (error) {
      next(error);
    }
  }
}

export const userController = new UserController();
