import { Request, Response, NextFunction } from 'express';
import { validationResult, body, param, query } from 'express-validator';
import { SPORT_CATEGORIES } from '../types';

export const validate = (req: Request, res: Response, next: NextFunction): void => {
  const errors = validationResult(req);
  if (!errors.isEmpty()) {
    res.status(400).json({
      success: false,
      error: 'Validation failed',
      details: errors.array(),
    });
    return;
  }
  next();
};

// Auth validation rules
export const registerValidation = [
  body('username')
    .trim()
    .isLength({ min: 3, max: 50 })
    .matches(/^[a-zA-Z0-9_]+$/)
    .withMessage('Username must be 3-50 characters and contain only letters, numbers, and underscores'),
  body('email').trim().isEmail().normalizeEmail().withMessage('Invalid email address'),
  body('password')
    .isLength({ min: 8 })
    .matches(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/)
    .withMessage('Password must be at least 8 characters with uppercase, lowercase, and number'),
  body('role').isIn(['athlete', 'follower']).withMessage('Role must be athlete or follower'),
  body('displayName').optional().trim().isLength({ max: 100 }),
  validate,
];

export const loginValidation = [
  body('username').trim().notEmpty().withMessage('Username is required'),
  body('password').notEmpty().withMessage('Password is required'),
  validate,
];

// Stream validation rules
export const createStreamValidation = [
  body('title').trim().isLength({ min: 1, max: 200 }).withMessage('Title must be 1-200 characters'),
  body('description').optional().trim().isLength({ max: 1000 }),
  body('sportCategory').isIn(SPORT_CATEGORIES).withMessage('Invalid sport category'),
  body('tags').optional().isArray().withMessage('Tags must be an array'),
  body('tags.*').optional().trim().isLength({ max: 50 }),
  validate,
];

export const updateStreamValidation = [
  body('title').optional().trim().isLength({ min: 1, max: 200 }),
  body('description').optional().trim().isLength({ max: 1000 }),
  body('sportCategory').optional().isIn(SPORT_CATEGORIES),
  body('tags').optional().isArray(),
  body('tags.*').optional().trim().isLength({ max: 50 }),
  validate,
];

// User profile validation
export const updateProfileValidation = [
  body('displayName').optional().trim().isLength({ max: 100 }),
  body('bio').optional().trim().isLength({ max: 500 }),
  body('sportCategories').optional().isArray(),
  body('sportCategories.*').optional().isIn(SPORT_CATEGORIES),
  validate,
];

// Pagination validation
export const paginationValidation = [
  query('page').optional().isInt({ min: 1 }).toInt(),
  query('limit').optional().isInt({ min: 1, max: 100 }).toInt(),
  validate,
];

// UUID validation
export const uuidValidation = [
  param('id').isUUID().withMessage('Invalid UUID format'),
  validate,
];
