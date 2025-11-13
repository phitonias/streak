import { Router } from 'express';
import { authController } from '../controllers/authController';
import { registerValidation, loginValidation } from '../middleware/validation';
import { authenticate } from '../middleware/auth';

const router = Router();

// Public routes
router.post('/register', registerValidation, authController.register.bind(authController));
router.post('/login', loginValidation, authController.login.bind(authController));
router.post('/refresh', authController.refreshToken.bind(authController));

// Protected routes
router.get('/me', authenticate, authController.getMe.bind(authController));
router.post('/logout', authenticate, authController.logout.bind(authController));

export default router;
