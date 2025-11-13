import { Server as SocketIOServer, Socket } from 'socket.io';
import { Server as HTTPServer } from 'http';
import jwt from 'jsonwebtoken';
import { config } from '../config';
import { JWTPayload, SocketEvent, ChatMessage } from '../types';
import { chatService } from './chatService';
import { streamService } from './streamService';
import { logger } from '../utils/logger';

export class SocketService {
  private io: SocketIOServer;

  constructor(httpServer: HTTPServer) {
    this.io = new SocketIOServer(httpServer, {
      cors: {
        origin: config.cors.origin,
        methods: ['GET', 'POST'],
        credentials: true,
      },
      transports: ['websocket', 'polling'],
    });

    this.setupMiddleware();
    this.setupEventHandlers();
  }

  private setupMiddleware(): void {
    // Authentication middleware
    this.io.use((socket: Socket, next) => {
      try {
        const token = socket.handshake.auth.token;

        if (!token) {
          logger.warn('Socket connection attempt without token');
          return next(new Error('Authentication required'));
        }

        const decoded = jwt.verify(token, config.jwt.secret) as JWTPayload;
        (socket as any).user = decoded;
        next();
      } catch (error) {
        logger.error('Socket authentication error:', error);
        next(new Error('Authentication failed'));
      }
    });
  }

  private setupEventHandlers(): void {
    this.io.on('connection', (socket: Socket) => {
      const user = (socket as any).user as JWTPayload;
      logger.info(`Socket connected: ${socket.id} (user: ${user.username})`);

      // Join stream room
      socket.on(SocketEvent.JOIN_STREAM, async (streamId: string) => {
        try {
          await socket.join(streamId);

          // Add viewer to stream
          const viewerCount = await streamService.addViewer(streamId, user.userId);

          // Notify user they joined
          socket.emit(SocketEvent.USER_JOINED, { streamId });

          // Broadcast new viewer count
          this.io.to(streamId).emit(SocketEvent.VIEWER_COUNT_UPDATE, {
            streamId,
            viewerCount,
          });

          // Send recent chat history
          const recentMessages = await chatService.getRecentMessages(streamId);
          socket.emit('chat_history', { streamId, messages: recentMessages });

          logger.info(`User ${user.username} joined stream ${streamId}`);
        } catch (error) {
          logger.error('Error joining stream:', error);
          socket.emit(SocketEvent.ERROR, { message: 'Failed to join stream' });
        }
      });

      // Leave stream room
      socket.on(SocketEvent.LEAVE_STREAM, async (streamId: string) => {
        try {
          await socket.leave(streamId);

          // Remove viewer from stream
          const viewerCount = await streamService.removeViewer(streamId, user.userId);

          // Broadcast new viewer count
          this.io.to(streamId).emit(SocketEvent.VIEWER_COUNT_UPDATE, {
            streamId,
            viewerCount,
          });

          logger.info(`User ${user.username} left stream ${streamId}`);
        } catch (error) {
          logger.error('Error leaving stream:', error);
        }
      });

      // Send chat message
      socket.on(SocketEvent.SEND_MESSAGE, async (data: { streamId: string; message: string }) => {
        try {
          const { streamId, message } = data;

          if (!message || message.trim().length === 0) {
            return;
          }

          if (message.length > 500) {
            socket.emit(SocketEvent.ERROR, { message: 'Message too long (max 500 characters)' });
            return;
          }

          const chatMessage: ChatMessage = {
            stream_id: streamId,
            user_id: user.userId,
            username: user.username,
            message: message.trim(),
            timestamp: new Date(),
            is_deleted: false,
          };

          // Save message to database
          await chatService.saveMessage(chatMessage);

          // Broadcast message to all users in the stream
          this.io.to(streamId).emit(SocketEvent.MESSAGE, chatMessage);

          logger.debug(`Message sent in stream ${streamId} by ${user.username}`);
        } catch (error) {
          logger.error('Error sending message:', error);
          socket.emit(SocketEvent.ERROR, { message: 'Failed to send message' });
        }
      });

      // Handle disconnection
      socket.on('disconnect', async () => {
        try {
          // Get all rooms the user was in
          const rooms = Array.from(socket.rooms).filter(room => room !== socket.id);

          // Remove user from all streams
          for (const streamId of rooms) {
            const viewerCount = await streamService.removeViewer(streamId, user.userId);
            this.io.to(streamId).emit(SocketEvent.VIEWER_COUNT_UPDATE, {
              streamId,
              viewerCount,
            });
          }

          logger.info(`Socket disconnected: ${socket.id} (user: ${user.username})`);
        } catch (error) {
          logger.error('Error handling disconnect:', error);
        }
      });
    });
  }

  // Method for broadcasting stream events from REST API
  public broadcastStreamStarted(streamId: string): void {
    this.io.emit(SocketEvent.STREAM_STARTED, { streamId });
    logger.info(`Broadcast: Stream ${streamId} started`);
  }

  public broadcastStreamEnded(streamId: string): void {
    this.io.to(streamId).emit(SocketEvent.STREAM_ENDED, { streamId });
    logger.info(`Broadcast: Stream ${streamId} ended`);
  }

  public getIO(): SocketIOServer {
    return this.io;
  }
}
