const http = require('http');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

const DIR = __dirname;
const workerScript = fs.readFileSync(path.join(DIR, 'worker.js'), 'utf8');
const wasmBuffer = fs.readFileSync(path.join(DIR, 'sha3.wasm'));

const headers = (token = '') => ({
  'host': 'chat.deepseek.com',
  'x-client-platform': 'android',
  'x-client-version': '2.3.1',
  'x-client-locale': 'id',
  'x-client-bundle-id': 'com.deepseek.chat',
  'x-rangers-id': '7700431188367789825',
  'x-client-timezone-offset': '25200',
  'user-agent': 'DeepSeek/2.3.1 Android/33',
  'authorization': token ? `Bearer ${token}` : '',
  'accept': 'application/json',
  'accept-charset': 'UTF-8',
  'content-type': 'application/json',
  'accept-encoding': 'gzip'
});

function solveChallenge(challengeData) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('PoW Solver Timeout')), 30000);

    class WasmResponse {
      constructor(buf) { this.buf = buf; this.ok = true; this.status = 200; this.headers = { get: () => 'application/wasm' }; }
      async arrayBuffer() { return this.buf; }
    }

    const contextObject = {
      console: { log: () => {} },
      setTimeout, clearTimeout, setInterval, clearInterval,
      TextEncoder, TextDecoder, URL,
      Response: WasmResponse,
      location: { href: 'https://static.deepseek.com/chat/static/33614.25c7f8f220.js', toString() { return this.href; } },
      WebAssembly: {
        ...WebAssembly,
        instantiateStreaming: async (source, imports) => WebAssembly.instantiate(wasmBuffer, imports)
      },
      fetch: async (target) => {
        if (String(target).includes('wasm')) return new WasmResponse(wasmBuffer);
        throw new Error('Network call blocked inside sandbox');
      },
      postMessage: (payload) => {
        if (!payload) return;
        clearTimeout(timer);
        if (payload.type === 'pow-answer') {
          const result = {
            algorithm: challengeData.algorithm,
            challenge: challengeData.challenge,
            salt: challengeData.salt,
            answer: payload.answer.answer,
            signature: challengeData.signature,
            target_path: challengeData.target_path
          };
          resolve(Buffer.from(JSON.stringify(result)).toString('base64'));
        } else {
          reject(new Error('PoW execution failed'));
        }
      },
      onmessage: null
    };

    contextObject.self = contextObject;
    contextObject.window = contextObject;
    contextObject.globalThis = contextObject;

    const context = vm.createContext(contextObject);
    vm.runInContext(workerScript, context);

    const msgHandler = contextObject.onmessage || contextObject.self?.onmessage;
    if (!msgHandler) {
      clearTimeout(timer);
      return reject(new Error('PoW worker message handler not found'));
    }

    msgHandler({
      data: {
        type: 'pow-challenge',
        challenge: {
          algorithm: challengeData.algorithm,
          challenge: challengeData.challenge,
          salt: challengeData.salt,
          difficulty: challengeData.difficulty,
          signature: challengeData.signature,
          expireAt: challengeData.expire_at
        }
      }
    });
  });
}

async function fetchPowToken(token, targetPath) {
  const response = await fetch('https://chat.deepseek.com/api/v0/chat/create_pow_challenge', {
    method: 'POST',
    headers: headers(token),
    body: JSON.stringify({ target_path: targetPath })
  });
  const json = await response.json();
  if (json.code !== 0 || !json.data?.biz_data?.challenge) {
    throw new Error(json.msg || 'Failed to create PoW challenge');
  }
  return solveChallenge(json.data.biz_data.challenge);
}

async function loginUser(email, password) {
  const deviceId = 'BUelgEoBdkHyhwE8q/4YOodITQ1Ef99t7Y5KAR4' + Math.random().toString(36).substring(2, 10);
  const response = await fetch('https://chat.deepseek.com/api/v0/users/login', {
    method: 'POST',
    headers: headers(),
    body: JSON.stringify({ email, password, device_id: deviceId, os: 'android' })
  });

  const json = await response.json();
  if (json.code !== 0) throw new Error(json.msg || 'Login failed');
  const user = json.data?.biz_data?.user;
  if (!user || !user.token) throw new Error('Token not returned from login');
  return {
    token: user.token,
    id: user.id || '',
    email: user.email || email
  };
}

async function uploadFile(token, filePath) {
  if (!fs.existsSync(filePath)) throw new Error('File not found: ' + filePath);
  const fileBuffer = fs.readFileSync(filePath);
  const fileName = path.basename(filePath);
  const fileSize = fileBuffer.length;

  const powToken = await fetchPowToken(token, '/api/v0/file/upload_file');
  const boundary = `----${Date.now().toString(16)}-${Math.random().toString(16).slice(2)}`;

  const bodyParts = [
    `--${boundary}\r\n`,
    `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`,
    `Content-Type: application/octet-stream\r\n`,
    `\r\n`
  ];

  const header = Buffer.from(bodyParts.join(''));
  const footer = Buffer.from(`\r\n--${boundary}--\r\n`);
  const body = Buffer.concat([header, fileBuffer, footer]);

  const uploadHeaders = {
    'host': 'chat.deepseek.com',
    'x-client-platform': 'android',
    'x-client-version': '2.3.1',
    'x-client-locale': 'id',
    'x-client-bundle-id': 'com.deepseek.chat',
    'x-rangers-id': '7700431188367789825',
    'x-client-timezone-offset': '25200',
    'user-agent': 'DeepSeek/2.3.1 Android/33',
    'authorization': `Bearer ${token}`,
    'accept': 'application/json',
    'accept-charset': 'UTF-8',
    'accept-encoding': 'gzip',
    'x-ds-pow-response': powToken,
    'x-thinking-enabled': '0',
    'x-file-size': String(fileSize),
    'x-model-type': 'vision',
    'content-type': `multipart/form-data; boundary=${boundary}`,
    'content-length': String(body.length)
  };

  const uploadRes = await fetch('https://chat.deepseek.com/api/v0/file/upload_file', {
    method: 'POST',
    headers: uploadHeaders,
    body
  });

  const uploadJson = await uploadRes.json();
  if (uploadJson.code !== 0) throw new Error(uploadJson.msg || 'Upload failed');

  const fileId = uploadJson.data.biz_data.id;

  const maxRetry = 20;
  for (let i = 0; i < maxRetry; i++) {
    await new Promise(r => setTimeout(r, 1500));
    const pollRes = await fetch(`https://chat.deepseek.com/api/v0/file/fetch_files?file_ids=${fileId}`, {
      headers: headers(token)
    });
    const pollJson = await pollRes.json();
    const file = pollJson.data?.biz_data?.files?.[0];
    if (file?.status === 'SUCCESS') {
      return { id: fileId, name: fileName, size: fileSize, status: 'SUCCESS' };
    }
    if (file?.status === 'FAILED') {
      throw new Error(`File processing failed: ${file.error_code}`);
    }
  }

  throw new Error('File processing timeout');
}

// Start HTTP Daemon
const PORT = parseInt(process.env.POW_PORT || '18833', 10);
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);

  if (req.method === 'GET' && url.pathname === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    return res.end(JSON.stringify({ status: 'ok', service: 'deepseek-pow-solver' }));
  }

  if (req.method === 'POST') {
    let bodyText = '';
    req.on('data', chunk => { bodyText += chunk; });
    req.on('end', async () => {
      let body = {};
      try { body = JSON.parse(bodyText || '{}'); } catch (_) {}

      if (url.pathname === '/pow') {
        const token = body.token || '';
        const targetPath = body.target_path || '/api/v0/chat/completion';
        try {
          const powToken = await fetchPowToken(token, targetPath);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, pow_token: powToken }));
        } catch (err) {
          res.writeHead(500, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      if (url.pathname === '/login') {
        const email = body.email || '';
        const password = body.password || '';
        if (!email || !password) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Email and password required' }));
        }
        try {
          const userInfo = await loginUser(email, password);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, ...userInfo }));
        } catch (err) {
          res.writeHead(401, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      if (url.pathname === '/upload') {
        const token = body.token || '';
        const filePath = body.file_path || '';
        if (!token || !filePath) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Token and file_path required' }));
        }
        try {
          const uploaded = await uploadFile(token, filePath);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, file: uploaded }));
        } catch (err) {
          res.writeHead(500, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Not found' }));
    });
    return;
  }

  res.writeHead(405, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Method not allowed' }));
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`[+] DeepSeek PoW Daemon listening on http://127.0.0.1:${PORT}`);
});
