import { getMongoDb } from '../config/database';
import { ChatMessage } from '../types';
import { logger } from '../utils/logger';

export class ChatService {
  async saveMessage(message: ChatMessage): Promise<void> {
    try {
      const db = getMongoDb();
      await db.collection('messages').insertOne({
        ...message,
        timestamp: new Date(),
        is_deleted: false,
      });
    } catch (error) {
      logger.error('Error saving chat message:', error);
    }
  }

  async getRecentMessages(streamId: string, limit: number = 50): Promise<ChatMessage[]> {
    try {
      const db = getMongoDb();
      const messages = await db
        .collection('messages')
        .find({ stream_id: streamId, is_deleted: false })
        .sort({ timestamp: -1 })
        .limit(limit)
        .toArray();

      return messages.reverse();
    } catch (error) {
      logger.error('Error fetching chat messages:', error);
      return [];
    }
  }

  async deleteMessage(messageId: string): Promise<void> {
    try {
      const db = getMongoDb();
      await db.collection('messages').updateOne(
        { _id: messageId },
        { $set: { is_deleted: true } }
      );
    } catch (error) {
      logger.error('Error deleting chat message:', error);
    }
  }

  async getMessageCount(streamId: string): Promise<number> {
    try {
      const db = getMongoDb();
      const count = await db
        .collection('messages')
        .countDocuments({ stream_id: streamId, is_deleted: false });
      return count;
    } catch (error) {
      logger.error('Error getting message count:', error);
      return 0;
    }
  }
}

export const chatService = new ChatService();
