import { Router } from 'express';
import { userController } from '../controllers/userController';
import { authenticate } from '../middleware/auth';
import { updateProfileValidation, paginationValidation, uuidValidation } from '../middleware/validation';

const router = Router();

// Get user by ID
router.get('/:id', uuidValidation, userController.getUserById.bind(userController));

// Get user by username
router.get('/username/:username', userController.getUserByUsername.bind(userController));

// Update profile (authenticated)
router.put('/profile', authenticate, updateProfileValidation, userController.updateProfile.bind(userController));

// Follow/unfollow
router.post('/:id/follow', authenticate, uuidValidation, userController.followUser.bind(userController));
router.delete('/:id/follow', authenticate, uuidValidation, userController.unfollowUser.bind(userController));

// Get followers/following
router.get('/:id/followers', paginationValidation, userController.getFollowers.bind(userController));
router.get('/:id/following', paginationValidation, userController.getFollowing.bind(userController));

// Check if following
router.get('/:id/is-following', authenticate, uuidValidation, userController.checkFollowing.bind(userController));

export default router;
