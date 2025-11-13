import dotenv from 'dotenv';

dotenv.config();

export const config = {
  env: process.env.NODE_ENV || 'development',
  rtmpPort: parseInt(process.env.RTMP_PORT || '1935', 10),
  httpPort: parseInt(process.env.HTTP_PORT || '8080', 10),
  redisUrl: process.env.REDIS_URL || 'redis://localhost:6379',
  apiUrl: process.env.API_URL || 'http://localhost:3001',
  hlsOutputPath: process.env.HLS_OUTPUT_PATH || './hls',
};
