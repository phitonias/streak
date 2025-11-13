import NodeMediaServer from 'node-media-server';
import express from 'express';
import path from 'path';
import fs from 'fs';
import { config } from './config';
import { logger } from './utils/logger';

// Ensure HLS directory exists
if (!fs.existsSync(config.hlsOutputPath)) {
  fs.mkdirSync(config.hlsOutputPath, { recursive: true });
  logger.info(`Created HLS output directory: ${config.hlsOutputPath}`);
}

// Node Media Server configuration
const nmsConfig = {
  rtmp: {
    port: config.rtmpPort,
    chunk_size: 60000,
    gop_cache: true,
    ping: 30,
    ping_timeout: 60,
  },
  http: {
    port: config.httpPort,
    mediaroot: config.hlsOutputPath,
    allow_origin: '*',
  },
  trans: {
    ffmpeg: '/usr/bin/ffmpeg',
    tasks: [
      {
        app: 'live',
        hls: true,
        hlsFlags: '[hls_time=2:hls_list_size=3:hls_flags=delete_segments]',
        hlsKeep: true, // Keep segments for a while
        dash: false,
      },
    ],
  },
  auth: {
    play: false,
    publish: true, // Require authentication for publishing
    secret: 'streak_secret_key', // This should be environment variable in production
  },
};

// Create Node Media Server
const nms = new NodeMediaServer(nmsConfig);

// Event handlers
nms.on('preConnect', (id, args) => {
  logger.info(`[PreConnect] id=${id} args=${JSON.stringify(args)}`);
});

nms.on('postConnect', (id, args) => {
  logger.info(`[PostConnect] id=${id} args=${JSON.stringify(args)}`);
});

nms.on('doneConnect', (id, args) => {
  logger.info(`[DoneConnect] id=${id} args=${JSON.stringify(args)}`);
});

nms.on('prePublish', async (id, StreamPath, args) => {
  logger.info(`[PrePublish] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);

  // Extract stream key from path (format: /live/STREAM_KEY)
  const streamKey = StreamPath.split('/').pop();

  // TODO: Validate stream key with backend API
  // For now, we'll allow all streams for MVP
  logger.info(`Stream key: ${streamKey}`);
});

nms.on('postPublish', async (id, StreamPath, args) => {
  logger.info(`[PostPublish] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);

  const streamKey = StreamPath.split('/').pop();

  // TODO: Notify backend that stream has started
  logger.info(`Stream started: ${streamKey}`);
});

nms.on('donePublish', async (id, StreamPath, args) => {
  logger.info(`[DonePublish] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);

  const streamKey = StreamPath.split('/').pop();

  // TODO: Notify backend that stream has ended
  logger.info(`Stream ended: ${streamKey}`);
});

nms.on('prePlay', (id, StreamPath, args) => {
  logger.info(`[PrePlay] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);
});

nms.on('postPlay', (id, StreamPath, args) => {
  logger.info(`[PostPlay] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);
});

nms.on('donePlay', (id, StreamPath, args) => {
  logger.info(`[DonePlay] id=${id} StreamPath=${StreamPath} args=${JSON.stringify(args)}`);
});

// Create Express server for HLS delivery
const app = express();

// CORS headers for HLS
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Methods', 'GET, HEAD, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Range');
  next();
});

// Serve HLS files
app.use('/hls', express.static(config.hlsOutputPath, {
  setHeaders: (res, filePath) => {
    if (filePath.endsWith('.m3u8')) {
      res.setHeader('Content-Type', 'application/vnd.apple.mpegurl');
      res.setHeader('Cache-Control', 'no-cache, no-store, must-revalidate');
    } else if (filePath.endsWith('.ts')) {
      res.setHeader('Content-Type', 'video/mp2t');
      res.setHeader('Cache-Control', 'public, max-age=3');
    }
  },
}));

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    uptime: process.uptime(),
    timestamp: new Date().toISOString(),
  });
});

// List active streams
app.get('/streams', (req, res) => {
  const streams = nms.getSession() || [];
  res.json({
    success: true,
    data: {
      count: streams.length,
      streams,
    },
  });
});

// Start servers
try {
  nms.run();
  logger.info(`🎥 RTMP server started on port ${config.rtmpPort}`);
  logger.info(`📡 HTTP server started on port ${config.httpPort}`);
  logger.info(`📁 HLS output path: ${config.hlsOutputPath}`);
  logger.info(`Environment: ${config.env}`);

  logger.info(`
╔════════════════════════════════════════════════════════════╗
║           Streak Streaming Server Running                  ║
╠════════════════════════════════════════════════════════════╣
║  RTMP Ingest: rtmp://localhost:${config.rtmpPort}/live              ║
║  HLS Delivery: http://localhost:${config.httpPort}/hls              ║
║                                                            ║
║  Example OBS Settings:                                     ║
║    Server: rtmp://localhost:${config.rtmpPort}/live                 ║
║    Stream Key: {your_stream_key}                          ║
║                                                            ║
║  Watch Stream:                                             ║
║    http://localhost:${config.httpPort}/hls/{stream_key}/index.m3u8  ║
╚════════════════════════════════════════════════════════════╝
  `);
} catch (error) {
  logger.error('Failed to start streaming server:', error);
  process.exit(1);
}

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM signal received, shutting down gracefully...');
  nms.stop();
  process.exit(0);
});

process.on('SIGINT', () => {
  logger.info('SIGINT signal received, shutting down gracefully...');
  nms.stop();
  process.exit(0);
});

export default nms;
