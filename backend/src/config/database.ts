import { Pool } from 'pg';
import { createClient } from 'redis';
import { MongoClient } from 'mongodb';
import { config } from './index';
import { logger } from '../utils/logger';

// PostgreSQL connection pool
export const pgPool = new Pool({
  connectionString: config.database.url,
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

pgPool.on('error', (err) => {
  logger.error('Unexpected PostgreSQL error:', err);
});

// Redis client
export const redisClient = createClient({
  url: config.redis.url,
});

redisClient.on('error', (err) => {
  logger.error('Redis Client Error:', err);
});

redisClient.on('connect', () => {
  logger.info('Redis client connected');
});

// MongoDB client
export const mongoClient = new MongoClient(config.mongodb.url);
let mongoDb: any = null;

export async function connectDatabases(): Promise<void> {
  try {
    // Test PostgreSQL connection
    const pgClient = await pgPool.connect();
    logger.info('PostgreSQL connected successfully');
    pgClient.release();

    // Connect Redis
    await redisClient.connect();
    logger.info('Redis connected successfully');

    // Connect MongoDB
    await mongoClient.connect();
    mongoDb = mongoClient.db('streak_chat');
    logger.info('MongoDB connected successfully');

    // Create MongoDB indexes
    await mongoDb.collection('messages').createIndex({ stream_id: 1, timestamp: -1 });
    await mongoDb.collection('messages').createIndex({ user_id: 1 });
    logger.info('MongoDB indexes created');
  } catch (error) {
    logger.error('Database connection error:', error);
    throw error;
  }
}

export async function closeDatabases(): Promise<void> {
  try {
    await pgPool.end();
    await redisClient.quit();
    await mongoClient.close();
    logger.info('All database connections closed');
  } catch (error) {
    logger.error('Error closing database connections:', error);
  }
}

export function getMongoDb() {
  if (!mongoDb) {
    throw new Error('MongoDB not connected');
  }
  return mongoDb;
}

// Database query helper with error handling
export async function query(text: string, params?: any[]) {
  const start = Date.now();
  try {
    const res = await pgPool.query(text, params);
    const duration = Date.now() - start;
    logger.debug('Executed query', { text, duration, rows: res.rowCount });
    return res;
  } catch (error) {
    logger.error('Database query error:', { text, error });
    throw error;
  }
}
