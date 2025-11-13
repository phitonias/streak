import { Router } from 'express';
import { streamController } from '../controllers/streamController';
import { authenticate, requireRole } from '../middleware/auth';
import { UserRole } from '../types';
import {
  createStreamValidation,
  updateStreamValidation,
  paginationValidation,
  uuidValidation,
} from '../middleware/validation';

const router = Router();

// Public routes
router.get('/live', paginationValidation, streamController.getLiveStreams.bind(streamController));
router.get('/:id', uuidValidation, streamController.getStream.bind(streamController));
router.get('/athlete/:athleteId', paginationValidation, streamController.getAthleteStreams.bind(streamController));

// Protected routes (authenticated)
router.get('/following/live', authenticate, streamController.getFollowingStreams.bind(streamController));

// Athlete-only routes
router.post(
  '/',
  authenticate,
  requireRole([UserRole.ATHLETE]),
  createStreamValidation,
  streamController.createStream.bind(streamController)
);

router.put(
  '/:id',
  authenticate,
  requireRole([UserRole.ATHLETE]),
  uuidValidation,
  updateStreamValidation,
  streamController.updateStream.bind(streamController)
);

router.post(
  '/:id/start',
  authenticate,
  requireRole([UserRole.ATHLETE]),
  uuidValidation,
  streamController.startStream.bind(streamController)
);

router.post(
  '/:id/stop',
  authenticate,
  requireRole([UserRole.ATHLETE]),
  uuidValidation,
  streamController.stopStream.bind(streamController)
);

export default router;
