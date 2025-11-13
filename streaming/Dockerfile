FROM node:20-alpine

# Install FFmpeg for transcoding
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Create HLS output directory
RUN mkdir -p /app/hls

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy source code
COPY . .

# Build TypeScript
RUN npm run build

# Expose ports
EXPOSE 1935 8080

# Start application
CMD ["npm", "start"]
