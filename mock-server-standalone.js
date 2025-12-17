// Standalone Mock Chillstreams API - No Dependencies Required
// Uses Node.js built-in http module

import http from 'http';
import { URL } from 'url';

const PORT = 3000;
const TORBOX_API_KEY = process.env.TORBOX_API_KEY || '6748e313-ff29-4a26-80c1-34e8da4b79ee';

// Simple in-memory storage
const mockData = {
  poolKeys: [{
    id: 'pool-key-1',
    apiKey: TORBOX_API_KEY,
    currentSlots: 5,
    maxSlots: 35,
    isActive: true
  }],
  deviceAssignments: {},
  usageLogs: []
};

// Helper to parse JSON body
async function parseBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', chunk => body += chunk.toString());
    req.on('end', () => {
      try {
        resolve(body ? JSON.parse(body) : {});
      } catch (e) {
        reject(e);
      }
    });
    req.on('error', reject);
  });
}

// Create HTTP server
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const path = url.pathname;

  // Set CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  res.setHeader('Content-Type', 'application/json');

  // Handle OPTIONS
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  try {
    // Health check
    if (path === '/health' && req.method === 'GET') {
      res.writeHead(200);
      res.end(JSON.stringify({ status: 'ok', service: 'mock-chillstreams-api' }));
      return;
    }

    // Get pool key
    if (path === '/api/v1/internal/pool/get-key' && req.method === 'POST') {
      const body = await parseBody(req);
      const { userId, deviceId, action, hash } = body;

      console.log('\n📥 GET-KEY REQUEST:', {
        userId,
        deviceId: deviceId?.substring(0, 8) + '...',
        action,
        hash
      });

      if (!userId || !deviceId) {
        res.writeHead(400);
        res.end(JSON.stringify({ error: 'Missing required fields: userId, deviceId' }));
        return;
      }

      const userKey = `${userId}-${deviceId}`;
      const userDevices = Object.keys(mockData.deviceAssignments).filter(k =>
        k.startsWith(userId)
      ).length;

      // Device limit check
      if (!mockData.deviceAssignments[userKey] && userDevices >= 3) {
        console.log('⚠️  Device limit reached:', userDevices);
        res.writeHead(200);
        res.end(JSON.stringify({
          allowed: false,
          message: 'Maximum device limit (3) reached',
          deviceCount: userDevices
        }));
        return;
      }

      // Assign pool key
      const poolKey = mockData.poolKeys[0];
      mockData.deviceAssignments[userKey] = {
        poolKeyId: poolKey.id,
        lastUsed: new Date().toISOString()
      };

      console.log('✅ Pool key assigned:', {
        poolKeyId: poolKey.id,
        deviceCount: Object.keys(mockData.deviceAssignments).filter(k =>
          k.startsWith(userId)
        ).length
      });

      res.writeHead(200);
      res.end(JSON.stringify({
        allowed: true,
        poolKey: poolKey.apiKey,
        poolKeyId: poolKey.id,
        deviceCount: Object.keys(mockData.deviceAssignments).filter(k =>
          k.startsWith(userId)
        ).length,
        message: 'Pool key assigned successfully'
      }));
      return;
    }

    // Log usage
    if (path === '/api/v1/internal/pool/log-usage' && req.method === 'POST') {
      const body = await parseBody(req);
      const { userId, poolKeyId, action, hash, cached, bytes } = body;

      console.log('\n📊 USAGE LOG:', {
        userId,
        poolKeyId,
        action,
        hash,
        cached,
        bytes: bytes ? `${(bytes / 1024 / 1024).toFixed(2)} MB` : 'N/A'
      });

      if (!userId || !action) {
        res.writeHead(400);
        res.end(JSON.stringify({ error: 'Missing required fields: userId, action' }));
        return;
      }

      mockData.usageLogs.push({
        userId,
        poolKeyId,
        action,
        hash,
        cached,
        bytes,
        timestamp: new Date().toISOString()
      });

      console.log('✅ Usage logged successfully');
      res.writeHead(200);
      res.end(JSON.stringify({ success: true, message: 'Usage logged successfully' }));
      return;
    }

    // Get stats
    if (path === '/api/v1/internal/pool/stats' && req.method === 'GET') {
      res.writeHead(200);
      res.end(JSON.stringify({
        poolKeys: mockData.poolKeys.length,
        deviceAssignments: Object.keys(mockData.deviceAssignments).length,
        usageLogs: mockData.usageLogs.length,
        devices: mockData.deviceAssignments,
        recentUsage: mockData.usageLogs.slice(-10)
      }));
      return;
    }

    // 404
    res.writeHead(404);
    res.end(JSON.stringify({ error: 'Not found' }));

  } catch (error) {
    console.error('❌ Server error:', error);
    res.writeHead(500);
    res.end(JSON.stringify({ error: 'Internal server error' }));
  }
});

// Start server
server.listen(PORT, () => {
  console.log('\n🚀 Mock Chillstreams API Server');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`📡 Listening on: http://localhost:${PORT}`);
  console.log(`🔗 Endpoints:`);
  console.log(`   POST /api/v1/internal/pool/get-key`);
  console.log(`   POST /api/v1/internal/pool/log-usage`);
  console.log(`   GET  /api/v1/internal/pool/stats`);
  console.log(`   GET  /health`);
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`\n🔑 TorBox API Key: ${TORBOX_API_KEY.substring(0, 8)}...`);
  console.log('\n✅ Ready for chillproxy testing!\n');
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\n\n📊 Final Stats:');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`   Device Assignments: ${Object.keys(mockData.deviceAssignments).length}`);
  console.log(`   Usage Logs: ${mockData.usageLogs.length}`);
  console.log('\n👋 Shutting down mock server...\n');
  server.close();
  process.exit(0);
});

