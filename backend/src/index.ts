import express, { Application } from 'express';
import { createServer } from 'http';
import cors from 'cors';
import helmet from 'helmet';
import morgan from 'morgan';
import rateLimit from 'express-rate-limit';

import { config } from './config';
import { connectDatabases, closeDatabases } from './config/database';
import { logger } from './utils/logger';
import { errorHandler, notFoundHandler } from './middleware/errorHandler';
import { SocketService } from './services/socketService';

// Import routes
import authRoutes from './routes/authRoutes';
import userRoutes from './routes/userRoutes';
import streamRoutes from './routes/streamRoutes';

class App {
  public app: Application;
  private httpServer: any;
  private socketService: SocketService | null = null;

  constructor() {
    this.app = express();
    this.httpServer = createServer(this.app);
    this.setupMiddleware();
    this.setupRoutes();
    this.setupErrorHandlers();
  }

  private setupMiddleware(): void {
    // Security
    this.app.use(helmet());

    // CORS
    this.app.use(
      cors({
        origin: config.cors.origin,
        credentials: true,
      })
    );

    // Body parsing
    this.app.use(express.json());
    this.app.use(express.urlencoded({ extended: true }));

    // Logging
    if (config.env === 'development') {
      this.app.use(morgan('dev'));
    } else {
      this.app.use(morgan('combined', { stream: { write: (msg) => logger.info(msg.trim()) } }));
    }

    // Rate limiting
    const limiter = rateLimit({
      windowMs: config.rateLimit.windowMs,
      max: config.rateLimit.maxRequests,
      message: 'Too many requests from this IP, please try again later.',
      standardHeaders: true,
      legacyHeaders: false,
    });

    this.app.use('/api/', limiter);

    // Health check
    this.app.get('/health', (req, res) => {
      res.status(200).json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        uptime: process.uptime(),
      });
    });
  }

  private setupRoutes(): void {
    // API routes
    this.app.use('/api/auth', authRoutes);
    this.app.use('/api/users', userRoutes);
    this.app.use('/api/streams', streamRoutes);

    // Root endpoint
    this.app.get('/', (req, res) => {
      res.json({
        name: 'Streak API',
        version: '1.0.0',
        status: 'running',
      });
    });
  }

  private setupErrorHandlers(): void {
    // 404 handler
    this.app.use(notFoundHandler);

    // Global error handler
    this.app.use(errorHandler);
  }

  public async initialize(): Promise<void> {
    try {
      // Connect to databases
      await connectDatabases();
      logger.info('All databases connected successfully');

      // Initialize Socket.IO
      this.socketService = new SocketService(this.httpServer);
      logger.info('Socket.IO initialized');

      // Start server
      this.httpServer.listen(config.port, () => {
        logger.info(`🚀 Streak API server running on port ${config.port}`);
        logger.info(`Environment: ${config.env}`);
        logger.info(`CORS origin: ${config.cors.origin}`);
      });
    } catch (error) {
      logger.error('Failed to initialize application:', error);
      process.exit(1);
    }
  }

  public async shutdown(): Promise<void> {
    logger.info('Shutting down gracefully...');

    // Close HTTP server
    this.httpServer.close(() => {
      logger.info('HTTP server closed');
    });

    // Close database connections
    await closeDatabases();

    logger.info('Application shut down complete');
    process.exit(0);
  }
}

// Create and initialize app
const app = new App();

// Handle graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM signal received');
  app.shutdown();
});

process.on('SIGINT', () => {
  logger.info('SIGINT signal received');
  app.shutdown();
});

process.on('unhandledRejection', (reason, promise) => {
  logger.error('Unhandled Rejection at:', promise, 'reason:', reason);
});

process.on('uncaughtException', (error) => {
  logger.error('Uncaught Exception:', error);
  app.shutdown();
});

// Start the application
app.initialize();

export default app;
