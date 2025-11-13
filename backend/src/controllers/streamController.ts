import { Request, Response, NextFunction } from 'express';
import { streamService } from '../services/streamService';
import { AuthRequest } from '../types';
import { AppError } from '../middleware/errorHandler';

export class StreamController {
  async createStream(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      if (req.user.role !== 'athlete') {
        throw new AppError('Only athletes can create streams', 403);
      }

      const { title, description, sportCategory, tags } = req.body;

      const stream = await streamService.createStream({
        athleteId: req.user.userId,
        title,
        description,
        sportCategory,
        tags,
      });

      res.status(201).json({
        success: true,
        data: stream,
        message: 'Stream created successfully',
      });
    } catch (error) {
      next(error);
    }
  }

  async getStream(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { id } = req.params;
      const stream = await streamService.getStreamById(id);

      res.status(200).json({
        success: true,
        data: stream,
      });
    } catch (error) {
      next(error);
    }
  }

  async updateStream(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id } = req.params;
      const { title, description, sportCategory, tags, thumbnailUrl } = req.body;

      const stream = await streamService.updateStream(id, req.user.userId, {
        title,
        description,
        sportCategory,
        tags,
        thumbnailUrl,
      });

      res.status(200).json({
        success: true,
        data: stream,
        message: 'Stream updated successfully',
      });
    } catch (error) {
      next(error);
    }
  }

  async startStream(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id } = req.params;
      const stream = await streamService.startStream(req.user.userId, id);

      res.status(200).json({
        success: true,
        data: stream,
        message: 'Stream started successfully',
      });
    } catch (error) {
      next(error);
    }
  }

  async stopStream(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const { id } = req.params;
      await streamService.stopStream(req.user.userId, id);

      res.status(200).json({
        success: true,
        message: 'Stream stopped successfully',
      });
    } catch (error) {
      next(error);
    }
  }

  async getLiveStreams(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const page = parseInt(req.query.page as string) || 1;
      const limit = parseInt(req.query.limit as string) || 20;
      const category = req.query.category as string;

      const result = await streamService.getLiveStreams(page, limit, category);

      res.status(200).json({
        success: true,
        data: result.streams,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  async getAthleteStreams(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { athleteId } = req.params;
      const page = parseInt(req.query.page as string) || 1;
      const limit = parseInt(req.query.limit as string) || 20;

      const result = await streamService.getStreamsByAthlete(athleteId, page, limit);

      res.status(200).json({
        success: true,
        data: result.streams,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  async getFollowingStreams(req: AuthRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!req.user) {
        throw new AppError('Not authenticated', 401);
      }

      const streams = await streamService.getFollowingStreams(req.user.userId);

      res.status(200).json({
        success: true,
        data: streams,
      });
    } catch (error) {
      next(error);
    }
  }
}

export const streamController = new StreamController();
