// Mock Chillstreams API Server for Testing Chillproxy
// This simulates the Phase 2 endpoints that don't exist yet in Chillstreams

import express from 'express';
const app = express();
const PORT = 3000;

app.use(express.json());

// Simple in-memory storage for testing
const mockData = {
  // Mock pool keys (use your real TorBox API key here)
  poolKeys: [
    {
      id: 'pool-key-1',
      apiKey: process.env.TORBOX_API_KEY || 'YOUR_TORBOX_API_KEY_HERE', // Replace with real key
      currentSlots: 5,
      maxSlots: 35,
      isActive: true
    }
  ],

  // Track device assignments
  deviceAssignments: {},

  // Track usage logs
  usageLogs: []
};

// POST /api/v1/internal/pool/get-key
// Called by chillproxy to get assigned pool key for user
app.post('/api/v1/internal/pool/get-key', (req, res) => {
  const { userId, deviceId, action, hash } = req.body;

  console.log('\n📥 GET-KEY REQUEST:', {
    userId,
    deviceId: deviceId?.substring(0, 8) + '...',
    action,
    hash
  });

  // Validate request
  if (!userId || !deviceId) {
    console.log('❌ Missing userId or deviceId');
    return res.status(400).json({
      error: 'Missing required fields: userId, deviceId'
    });
  }

  // Check if user exists (mock validation)
  const userKey = `${userId}-${deviceId}`;

  // Count devices for this user
  const userDevices = Object.keys(mockData.deviceAssignments).filter(k =>
    k.startsWith(userId)
  ).length;

  // Mock device limit check (max 3 devices)
  if (!mockData.deviceAssignments[userKey] && userDevices >= 3) {
    console.log('⚠️  Device limit reached:', userDevices);
    return res.json({
      allowed: false,
      message: 'Maximum device limit (3) reached',
      deviceCount: userDevices
    });
  }

  // Assign pool key (round-robin for multiple keys)
  const poolKey = mockData.poolKeys[0]; // Use first key for simplicity

  // Track assignment
  mockData.deviceAssignments[userKey] = {
    poolKeyId: poolKey.id,
    lastUsed: new Date().toISOString()
  };

  console.log('✅ Pool key assigned:', {
    poolKeyId: poolKey.id,
    deviceCount: userDevices + (mockData.deviceAssignments[userKey] ? 0 : 1)
  });

  // Return success response
  res.json({
    allowed: true,
    poolKey: poolKey.apiKey,
    poolKeyId: poolKey.id,
    deviceCount: Object.keys(mockData.deviceAssignments).filter(k =>
      k.startsWith(userId)
    ).length,
    message: 'Pool key assigned successfully'
  });
});

// POST /api/v1/internal/pool/log-usage
// Called by chillproxy to log usage after serving stream
app.post('/api/v1/internal/pool/log-usage', (req, res) => {
  const { userId, poolKeyId, action, hash, cached, bytes } = req.body;

  console.log('\n📊 USAGE LOG:', {
    userId,
    poolKeyId,
    action,
    hash,
    cached,
    bytes: bytes ? `${(bytes / 1024 / 1024).toFixed(2)} MB` : 'N/A'
  });

  // Validate request
  if (!userId || !action) {
    console.log('❌ Missing userId or action');
    return res.status(400).json({
      error: 'Missing required fields: userId, action'
    });
  }

  // Store usage log
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

  // Return success
  res.json({
    success: true,
    message: 'Usage logged successfully'
  });
});

// GET /api/v1/internal/pool/stats (Bonus: View stats)
app.get('/api/v1/internal/pool/stats', (req, res) => {
  res.json({
    poolKeys: mockData.poolKeys.length,
    deviceAssignments: Object.keys(mockData.deviceAssignments).length,
    usageLogs: mockData.usageLogs.length,
    devices: mockData.deviceAssignments,
    recentUsage: mockData.usageLogs.slice(-10)
  });
});

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'mock-chillstreams-api' });
});

// Start server
app.listen(PORT, () => {
  console.log('\n🚀 Mock Chillstreams API Server');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`📡 Listening on: http://localhost:${PORT}`);
  console.log(`🔗 Endpoints:`);
  console.log(`   POST /api/v1/internal/pool/get-key`);
  console.log(`   POST /api/v1/internal/pool/log-usage`);
  console.log(`   GET  /api/v1/internal/pool/stats`);
  console.log(`   GET  /health`);
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log('\n⚠️  IMPORTANT: Set your TorBox API key:');
  console.log('   export TORBOX_API_KEY=your_key_here');
  console.log('   Or edit mock-chillstreams-api.js line 12\n');
  console.log('✅ Ready for chillproxy testing!\n');
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\n\n📊 Final Stats:');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`   Device Assignments: ${Object.keys(mockData.deviceAssignments).length}`);
  console.log(`   Usage Logs: ${mockData.usageLogs.length}`);
  console.log('\n👋 Shutting down mock server...\n');
  process.exit(0);
});

