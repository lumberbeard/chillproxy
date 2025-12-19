const http = require('http');

// Database connection using HTTP to query endpoint
const pool = null;

const TEST_USER_ID = '3b94cb45-3f99-406e-9c40-ecce61a405cc';
const TEST_DEVICE_ID = 'test-device-' + Math.random().toString(36).substring(7);
const API_URL = 'http://localhost:3000';
const API_KEY = 'test_internal_key_phase3_2025';

async function test1_HealthEndpoint() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║ TEST 1: /api/v1/health endpoint responds                      ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  return new Promise((resolve) => {
    const req = http.get(`${API_URL}/api/v1/health`, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        if (res.statusCode === 200) {
          console.log('✅ PASSED');
          console.log(`   Status: ${res.statusCode}`);
          console.log(`   Response: ${data}`);
          resolve(true);
        } else {
          console.log(`❌ FAILED - Got status ${res.statusCode}`);
          resolve(false);
        }
      });
    });

    req.on('error', (err) => {
      console.log(`❌ FAILED - ${err.message}`);
      resolve(false);
    });

    req.setTimeout(5000);
  });
}

async function test2_GetPoolKeyEndpoint() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║ TEST 2: /api/v1/internal/pool/get-key returns pool key        ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  const body = JSON.stringify({
    userId: TEST_USER_ID,
    deviceId: TEST_DEVICE_ID,
    action: 'init',
    hash: 'test-hash-123'
  });

  return new Promise((resolve) => {
    const options = {
      hostname: 'localhost',
      port: 3000,
      path: '/api/v1/internal/pool/get-key',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
        'Authorization': `Bearer ${API_KEY}`
      }
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          if (res.statusCode === 200 && json.allowed && json.poolKey) {
            console.log('✅ PASSED');
            console.log(`   Status: ${res.statusCode}`);
            console.log(`   Response:`, JSON.stringify(json, null, 2));
            resolve({ passed: true, data: json });
          } else {
            console.log(`❌ FAILED - ${json.message || 'No pool key returned'}`);
            console.log(`   Response:`, JSON.stringify(json, null, 2));
            resolve({ passed: false, data: json });
          }
        } catch (e) {
          console.log(`❌ FAILED - ${e.message}`);
          resolve({ passed: false, data: null });
        }
      });
    });

    req.on('error', (err) => {
      console.log(`❌ FAILED - ${err.message}`);
      resolve({ passed: false, data: null });
    });

    req.write(body);
    req.end();
  });
}

async function test3_AssignmentCreated() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║ TEST 3: Assignment created in torbox_assignments table        ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  try {
    const result = await pool.query(
      'SELECT * FROM torbox_assignments WHERE user_id = $1 AND device_id = $2',
      [TEST_USER_ID, TEST_DEVICE_ID]
    );

    if (result.rows.length > 0) {
      console.log('✅ PASSED');
      console.log(`   Found ${result.rows.length} assignment(s)`);
      console.log('   Assignment:', JSON.stringify(result.rows[0], null, 2));
      return { passed: true, data: result.rows[0] };
    } else {
      console.log('❌ FAILED - No assignment found');
      console.log(`   Searched for user_id=${TEST_USER_ID}, device_id=${TEST_DEVICE_ID}`);
      return { passed: false, data: null };
    }
  } catch (error) {
    console.log(`❌ FAILED - ${error.message}`);
    return { passed: false, data: null };
  }
}

async function test4_UsageLogged() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║ TEST 4: Usage logged to torbox_usage_logs                     ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  try {
    const result = await pool.query(
      'SELECT * FROM torbox_usage_logs WHERE user_id = $1 ORDER BY timestamp DESC LIMIT 1',
      [TEST_USER_ID]
    );

    if (result.rows.length > 0) {
      console.log('✅ PASSED');
      console.log(`   Found ${result.rows.length} usage log(s)`);
      console.log('   Latest log:', JSON.stringify(result.rows[0], null, 2));
      return { passed: true, data: result.rows[0] };
    } else {
      console.log('❌ FAILED - No usage logs found');
      return { passed: false, data: null };
    }
  } catch (error) {
    console.log(`❌ FAILED - ${error.message}`);
    return { passed: false, data: null };
  }
}

async function test5_AssignmentReused() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║ TEST 5: Pool key assignment reused on second request          ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  const body = JSON.stringify({
    userId: TEST_USER_ID,
    deviceId: TEST_DEVICE_ID,
    action: 'check-cache',
    hash: 'another-hash-456'
  });

  return new Promise((resolve) => {
    const options = {
      hostname: 'localhost',
      port: 3000,
      path: '/api/v1/internal/pool/get-key',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
        'Authorization': `Bearer ${API_KEY}`
      }
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          if (res.statusCode === 200 && json.allowed && json.poolKey) {
            console.log('✅ PASSED');
            console.log(`   Second request returned same/valid pool key`);
            console.log(`   Device count: ${json.deviceCount}`);
            console.log(`   Response:`, JSON.stringify(json, null, 2));
            resolve({ passed: true, data: json });
          } else {
            console.log(`❌ FAILED - ${json.message || 'Second request failed'}`);
            resolve({ passed: false, data: json });
          }
        } catch (e) {
          console.log(`❌ FAILED - ${e.message}`);
          resolve({ passed: false, data: null });
        }
      });
    });

    req.on('error', (err) => {
      console.log(`❌ FAILED - ${err.message}`);
      resolve({ passed: false, data: null });
    });

    req.write(body);
    req.end();
  });
}

async function runAllTests() {
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║                    CHILLPROXY INTEGRATION TESTS                ║');
  console.log('║                          Phase 2 Validation                    ║');
  console.log('╚════════════════════════════════════════════════════════════════╝');

  console.log(`\nTest Configuration:`);
  console.log(`  API URL: ${API_URL}`);
  console.log(`  User ID: ${TEST_USER_ID}`);
  console.log(`  Device ID: ${TEST_DEVICE_ID}`);

  const results = {};

  results.test1 = await test1_HealthEndpoint();
  results.test2 = await test2_GetPoolKeyEndpoint();
  results.test3 = await test3_AssignmentCreated();
  results.test4 = await test4_UsageLogged();
  results.test5 = await test5_AssignmentReused();

  // Summary
  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║                         TEST SUMMARY                          ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  const passCount = Object.values(results).filter(r => r === true || (r && r.passed)).length;
  const totalCount = Object.keys(results).length;

  console.log(`${passCount}/${totalCount} tests passed\n`);

  console.log('Results:');
  console.log(`  [${results.test1 ? '✅' : '❌'}] Test 1: Health endpoint`);
  console.log(`  [${results.test2.passed ? '✅' : '❌'}] Test 2: Get pool key endpoint`);
  console.log(`  [${results.test3.passed ? '✅' : '❌'}] Test 3: Assignment created`);
  console.log(`  [${results.test4.passed ? '✅' : '❌'}] Test 4: Usage logged`);
  console.log(`  [${results.test5.passed ? '✅' : '❌'}] Test 5: Assignment reused`);

  console.log('\n' + '╚════════════════════════════════════════════════════════════════╝\n');

  await pool.end();

  process.exit(passCount === totalCount ? 0 : 1);
}

runAllTests().catch(err => {
  console.error('Test error:', err);
  process.exit(1);
});

