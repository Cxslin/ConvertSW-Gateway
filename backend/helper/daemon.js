const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const crypto = require('crypto');

const DIR = __dirname;
const workerScript = fs.readFileSync(path.join(DIR, 'worker.js'), 'utf8');
const wasmBuffer = fs.readFileSync(path.join(DIR, 'sha3.wasm'));
const accountsFile = path.join(DIR, '../../data/accounts.json');

/* =========================================================================
   1. DEEPSEEK POW SOLVER & ENGINE
   ========================================================================= */

const dsHeaders = (token = '') => ({
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

function solveDeepSeekChallenge(challengeData) {
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

async function fetchDeepSeekPowToken(token, targetPath) {
  const response = await fetch('https://chat.deepseek.com/api/v0/chat/create_pow_challenge', {
    method: 'POST',
    headers: dsHeaders(token),
    body: JSON.stringify({ target_path: targetPath })
  });
  const json = await response.json();
  if (json.code !== 0 || !json.data?.biz_data?.challenge) {
    throw new Error(json.msg || 'Failed to create PoW challenge');
  }
  return solveDeepSeekChallenge(json.data.biz_data.challenge);
}

async function loginDeepSeek(email, password) {
  const deviceId = 'BUelgEoBdkHyhwE8q/4YOodITQ1Ef99t7Y5KAR4' + Math.random().toString(36).substring(2, 10);
  const response = await fetch('https://chat.deepseek.com/api/v0/users/login', {
    method: 'POST',
    headers: dsHeaders(),
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

async function uploadDeepSeekFile(token, filePath) {
  if (!fs.existsSync(filePath)) throw new Error('File not found: ' + filePath);
  const fileBuffer = fs.readFileSync(filePath);
  const fileName = path.basename(filePath);
  const fileSize = fileBuffer.length;

  const powToken = await fetchDeepSeekPowToken(token, '/api/v0/file/upload_file');
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
      headers: dsHeaders(token)
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


/* =========================================================================
   2. MISTRAL AI ENGINE (LE CHAT MOBILE REVERSE)
   ========================================================================= */

const MISTRAL_HEADERS = {
  'User-Agent': 'le-chat-mobile/2.7.0 (build:20700421; os_name:ios; device_category:smartphone; device_model:iPhone 13 Pro; device_manufacturer:Apple)',
  'Accept': '*/*',
  'Content-Type': 'application/json'
};

function parseCookies(setCookieArr) {
  if (!setCookieArr || !setCookieArr.length) return {};
  const arr = Array.isArray(setCookieArr) ? setCookieArr : [setCookieArr];
  return Object.fromEntries(
    arr.map(c => {
      const eq = c.indexOf('=');
      return eq < 0 ? [] : [c.slice(0, eq).trim(), c.slice(eq + 1).split(';')[0].trim()];
    }).filter(e => e.length === 2)
  );
}

function cookieToString(jar) {
  return Object.entries(jar).map(([k, v]) => `${k}=${v}`).join('; ');
}

async function mistralPost(endpoint, body, cookie = '') {
  const hdrs = { ...MISTRAL_HEADERS };
  if (cookie) hdrs['Cookie'] = cookie;

  const response = await fetch('https://chat.mistral.ai' + endpoint, {
    method: 'POST',
    headers: hdrs,
    body: JSON.stringify(body)
  });

  const rawCookies = response.headers.getSetCookie ? response.headers.getSetCookie() : [];
  const data = await response.text();
  let json = null;
  try { json = JSON.parse(data); } catch (e) {}

  return { json, raw: data, cookies: parseCookies(rawCookies) };
}

async function mistralHandshake() {
  const { cookies } = await mistralPost('/api/trpc/event.sendEventToDatalake,event.sendEventToDatalake?batch=1',
    {
      "0": { "json": { "name": "app_downloaded", "properties": {} } },
      "1": {
        "json": {
          "name": "app_started",
          "properties": {
            "os": "iOS", "osVersion": "18.3.2",
            "deviceManufacturer": "Apple",
            "screenWidth": 430, "screenHeight": 932,
            "windowWidth": 430, "windowHeight": 932,
            "pixelRatio": 3, "fontScale": 1,
            "deviceColorScheme": "dark",
            "preferredLocale": "en-US",
            "permissions": { "notifications": "undetermined", "camera": "undetermined", "mediaLibrary": "denied" }
          }
        }
      }
    }
  );

  const cookie = cookieToString(cookies);
  await mistralPost('/api/trpc/user.acceptToS?batch=1', { "0": { "json": {} } }, cookie);
  return { cookie, identifier: crypto.randomUUID() };
}

async function mistralCreateRoom(text, session) {
  const { json } = await mistralPost('/api/trpc/message.newChat?batch=1',
    {
      "0": {
        "json": {
          "files": [],
          "content": [{ "type": "text", "text": text }],
          "transcriptionsMetadata": null, "agentId": null,
          "agentsApiAgentId": null, "features": ["beta-websearch"],
          "integrations": [], "libraries": [],
          "productType": "chat", "projectId": null,
          "incognito": null, "chatId": null,
          "parentId": null, "parentVersion": null
        },
        "meta": {
          "values": {
            "transcriptionsMetadata": ["undefined"], "agentId": ["undefined"],
            "agentsApiAgentId": ["undefined"], "projectId": ["undefined"],
            "incognito": ["undefined"], "chatId": ["undefined"],
            "parentId": ["undefined"], "parentVersion": ["undefined"]
          },
          "v": 1
        }
      }
    },
    session.cookie
  );

  if (!json || !json[0]?.result?.data?.json?.chatId) {
    throw new Error('Mistral createRoom failed');
  }
  return json[0].result.data.json.chatId;
}

async function executeMistralChat(prompt, opts = {}, onChunk = null) {
  let session = opts.session;
  if (!session || !session.cookie) {
    session = await mistralHandshake();
  }

  let chatId = opts.chat_id;
  const fresh = !chatId;
  if (!chatId) {
    chatId = await mistralCreateRoom(prompt, session);
  }

  const payload = {
    chatId,
    stableAnonymousIdentifier: session.identifier,
    platform: 'mobile',
    clientPromptData: {
      currentDate: new Date().toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' }),
      userTimezone: 'Asia/Jakarta'
    },
    shouldAwaitStreamBackgroundTasks: true,
    shouldUseMessagePatch: true,
    features: opts.web_search ? ['beta-websearch'] : [],
    integrations: [],
    libraries: [],
    mode: fresh ? 'start' : 'append'
  };

  if (!fresh) {
    payload.messageId = crypto.randomUUID();
    payload.messageInput = [{ type: 'text', text: prompt }];
    payload.messageFiles = [];
  } else {
    payload.disabledFeatures = ['memory-inference'];
  }

  const res = await fetch('https://chat.mistral.ai/api/chat', {
    method: 'POST',
    headers: {
      ...MISTRAL_HEADERS,
      'Cookie': session.cookie,
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify(payload)
  });

  if (!res.ok) {
    const errText = await res.text();
    throw new Error(`Mistral HTTP ${res.status}: ${errText.slice(0, 200)}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  let fullText = '';
  let messageId = null;
  const contentMap = {};

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;

    buf += decoder.decode(value, { stream: true });
    const lines = buf.split('\n');
    buf = lines.pop() || '';

    for (const raw of lines) {
      const line = raw.trim();
      if (!line || line === 'null') continue;

      const colonIdx = line.indexOf(':');
      if (colonIdx < 1 || isNaN(parseInt(line[0]))) continue;

      const jsonStr = line.slice(colonIdx + 1).trim();
      if (!jsonStr || jsonStr === 'null') continue;

      let parsed;
      try { parsed = JSON.parse(jsonStr); } catch { continue; }
      const data = parsed?.json;
      if (!data) continue;

      if (data.type === 'bootstrap') {
        chatId = data.chat?.id || chatId;
      }

      if (data.type === 'message' && data.patches) {
        const mid = data.messageId || 'msg';
        messageId = mid;

        for (const patch of data.patches) {
          const { op, path: ppath, value: val } = patch;

          if (op === 'append' && ppath && ppath.includes('/text')) {
            const chunk = typeof val === 'string' ? val : '';
            if (!contentMap[mid]) contentMap[mid] = '';
            contentMap[mid] += chunk;
            fullText += chunk;
            if (onChunk && chunk) onChunk(chunk);
          }

          if (op === 'replace' && ppath && /\/contentChunks\/\d+\/text$/.test(ppath)) {
            const chunk = typeof val === 'string' ? val : '';
            contentMap[mid] = chunk;
            fullText = chunk;
            if (onChunk && chunk) onChunk(chunk);
          }
        }
      }
    }
  }

  return {
    reply: fullText.trim(),
    chat_id: chatId,
    message_id: messageId,
    session
  };
}


/* =========================================================================
   3. CHATGPT ENGINE (SENTINEL POW & CONVERSATION)
   ========================================================================= */

const agent = new https.Agent({ keepAlive: true });

function chatgptRequest(url, { method = 'GET', headers = {}, body = null, stream = false } = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const opts = {
      hostname: u.hostname,
      path: u.pathname + u.search,
      method,
      headers: body ? { ...headers, 'content-length': Buffer.byteLength(body) } : headers,
      agent,
      maxHeaderSize: 1048576
    };

    const req = https.request(opts, res => {
      if (stream) return resolve({ res, headers: res.headers, status: res.statusCode });
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ text: Buffer.concat(chunks).toString(), headers: res.headers, status: res.statusCode }));
    });

    req.on('error', reject);
    if (body) req.write(body);
    req.end();
  });
}

function fnv1a(str) {
  let h = 2166136261;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h = Math.imul(h, 16777619) >>> 0;
  }
  h ^= h >>> 16;
  h = Math.imul(h, 2246822507) >>> 0;
  h ^= h >>> 13;
  h = Math.imul(h, 3266489909) >>> 0;
  h ^= h >>> 16;
  return (h >>> 0).toString(16).padStart(8, '0');
}

function encodeConfig(cfg) {
  return Buffer.from(JSON.stringify(cfg)).toString('base64');
}

function makeBrowserConfig(width, height, userAgent, buildNumber, lang) {
  return [
    width + height,
    String(new Date()),
    2172649472,
    0,
    userAgent,
    null,
    buildNumber,
    lang,
    `${lang},en`,
    0,
    'contacts\u2212[object ContactsManager]',
    '_reactListening',
    'User',
    performance.now(),
    crypto.randomUUID(),
    '',
    8,
    performance.timeOrigin,
    0, 0, 0, 0, 0, 0, 0
  ];
}

function computePow(seed, difficulty, cfg) {
  const start = performance.now();
  for (let i = 0; i < 500000; i++) {
    cfg[3] = i;
    cfg[9] = Math.round(performance.now() - start);
    const encoded = encodeConfig(cfg);
    if (fnv1a(seed + encoded).substring(0, difficulty.length) <= difficulty) {
      return 'gAAAAAB' + encoded + '~S';
    }
  }
  return 'wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4De';
}

async function chatgptStartSession(cookieStr = null) {
  const cookies = {};
  const userAgent = 'Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Mobile Safari/537.36';
  const oaiDid = crypto.randomUUID();
  const lang = 'id-ID';

  let buildNumber = null;
  const homeRes = await chatgptRequest('https://chatgpt.com/', {
    method: 'GET',
    headers: {
      'User-Agent': userAgent,
      'accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      'accept-language': `${lang},en-US;q=0.9,en;q=0.8`
    }
  });

  const html = homeRes.text || '';
  const match = html.match(/["']buildNumber["']\s*:\s*["']([^"']+)["']/) ||
                html.match(/prod-([a-f0-9]+)/);
  if (match && match[1]) buildNumber = match[1];

  if (cookieStr) {
    for (const pair of cookieStr.split(';')) {
      const idx = pair.indexOf('=');
      if (idx > 0) cookies[pair.slice(0, idx).trim()] = pair.slice(idx + 1).trim();
    }
  }

  const headers = {
    'User-Agent': userAgent,
    'accept': '*/*',
    'accept-language': `${lang},en-US;q=0.9,en;q=0.8`,
    'content-type': 'application/json',
    'OAI-Device-Id': oaiDid,
    'origin': 'https://chatgpt.com',
    'referer': 'https://chatgpt.com/',
    ...(Object.keys(cookies).length > 0 ? { 'cookie': Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; ') } : {})
  };

  return {
    cookies,
    headers,
    userAgent,
    oaiDid,
    buildNumber,
    lang,
    screenWidth: 423,
    screenHeight: 965
  };
}

async function chatgptGenerateSentinelTokens(auth) {
  const initCfg = makeBrowserConfig(auth.screenWidth, auth.screenHeight, auth.userAgent, auth.buildNumber, auth.lang);
  initCfg[3] = 1;
  initCfg[9] = 0;
  const initToken = 'gAAAAAC' + encodeConfig(initCfg);

  const prepareRes = await chatgptRequest('https://chatgpt.com/backend-anon/sentinel/chat-requirements/prepare', {
    method: 'POST',
    headers: auth.headers,
    body: JSON.stringify({ p: initToken })
  });
  const prepareData = JSON.parse(prepareRes.text || '{}');

  let pow = null;
  if (prepareData.proofofwork?.required) {
    pow = computePow(
      prepareData.proofofwork.seed,
      prepareData.proofofwork.difficulty,
      makeBrowserConfig(auth.screenWidth, auth.screenHeight, auth.userAgent, auth.buildNumber, auth.lang)
    );
  }

  const turnstile = crypto.randomBytes(Math.floor((2256 / 4) * 3)).toString('base64').slice(0, 2256);
  const finalizeBody = { prepare_token: prepareData.prepare_token ?? '' };
  if (pow) finalizeBody.proofofwork = pow;
  if (turnstile) finalizeBody.turnstile = turnstile;

  const finalizeRes = await chatgptRequest('https://chatgpt.com/backend-anon/sentinel/chat-requirements/finalize', {
    method: 'POST',
    headers: auth.headers,
    body: JSON.stringify(finalizeBody)
  });
  const finalizeData = JSON.parse(finalizeRes.text || '{}');

  return {
    pow,
    turnstile,
    chatRequirementsToken: finalizeData.token ?? null
  };
}

async function chatgptGetConduitToken(message, msgId, auth, opts = {}) {
  const { conversationId, parentMsgId, webSearch } = opts;
  const parentMessageId = parentMsgId || 'client-created-root';

  const body = {
    action: 'next',
    fork_from_shared_post: false,
    parent_message_id: parentMessageId,
    model: 'auto',
    timezone_offset_min: new Date().getTimezoneOffset(),
    timezone: 'Asia/Jakarta',
    conversation_mode: { kind: 'primary_assistant' },
    system_hints: webSearch ? ['search'] : [],
    supports_buffering: true,
    supported_encodings: ['v1'],
    partial_query: {
      id: msgId,
      author: { role: 'user' },
      content: { content_type: 'text', parts: [message] }
    },
    client_contextual_info: { app_name: 'chatgpt.com' }
  };

  if (conversationId) body.conversation_id = conversationId;

  const res = await chatgptRequest('https://chatgpt.com/backend-anon/f/conversation/prepare', {
    method: 'POST',
    headers: { ...auth.headers, 'X-Conduit-Token': 'no-token' },
    body: JSON.stringify(body)
  });
  const data = JSON.parse(res.text || '{}');
  return data.token || data.conduit_token;
}

async function executeChatGPTChat(prompt, opts = {}, onChunk = null) {
  let auth = opts.auth;
  if (!auth) {
    auth = await chatgptStartSession(opts.cookie || null);
  }

  const { webSearch = false } = opts;
  const msgId = crypto.randomUUID();
  const parentMessageId = opts.parent_message_id || 'client-created-root';
  const conversationId = opts.chat_id || null;

  const [tokens, conduitToken] = await Promise.all([
    chatgptGenerateSentinelTokens(auth),
    chatgptGetConduitToken(prompt, msgId, auth, { conversationId, parentMsgId: parentMessageId, webSearch })
  ]);

  const body = {
    action: 'next',
    messages: [{
      id: msgId,
      author: { role: 'user' },
      create_time: Date.now() / 1000,
      content: { content_type: 'text', parts: [prompt] },
      metadata: {
        selected_github_repos: [],
        selected_all_github_repos: false,
        serialization_metadata: { custom_symbol_offsets: [] },
        ...(webSearch ? { system_hints: ['search'] } : {})
      }
    }],
    parent_message_id: parentMessageId,
    model: 'auto',
    timezone_offset_min: new Date().getTimezoneOffset(),
    timezone: 'Asia/Jakarta',
    conversation_mode: { kind: 'primary_assistant' },
    enable_message_followups: true,
    system_hints: webSearch ? ['search'] : [],
    supports_buffering: true,
    supported_encodings: ['v1'],
    client_contextual_info: {
      is_dark_mode: true,
      time_since_loaded: 10,
      page_height: 845,
      page_width: 423,
      pixel_ratio: 1.7,
      screen_height: auth.screenHeight,
      screen_width: auth.screenWidth,
      app_name: 'chatgpt.com'
    },
    force_parallel_switch: 'auto'
  };

  if (conversationId) body.conversation_id = conversationId;
  if (webSearch) {
    body.force_use_search = true;
    body.client_reported_search_source = 'conversation_composer_web_icon';
  }

  const headers = {
    ...auth.headers,
    'accept': 'text/event-stream',
    'OAI-Language': auth.lang,
    'OpenAI-Sentinel-Chat-Requirements-Token': tokens.chatRequirementsToken,
    'OpenAI-Sentinel-Turnstile-Token': tokens.turnstile,
    'OpenAI-Sentinel-Proof-Token': tokens.pow,
    'X-Conduit-Token': conduitToken
  };

  const { res } = await chatgptRequest('https://chatgpt.com/backend-anon/f/conversation', {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
    stream: true
  });

  return new Promise((resolve, reject) => {
    let fullText = '';
    let model = null;
    let convId = null;
    let assistantMsgId = null;
    let buf = '';

    res.on('data', chunk => {
      buf += chunk.toString('utf8');
      const events = buf.split('\n\n');
      buf = events.pop() || '';

      for (const event of events) {
        const lines = event.split('\n');
        for (const line of lines) {
          if (!line.startsWith('data:')) continue;
          const raw = line.slice(5).trim();
          if (!raw || raw === '[DONE]') continue;
          let json;
          try { json = JSON.parse(raw); } catch { continue; }

          if (json.conversation_id) convId = json.conversation_id;
          else if (json.v?.conversation_id) convId = json.v.conversation_id;

          if (json.v && !Array.isArray(json.v) && json.v.message?.author?.role === 'assistant' && json.v.message?.id) {
            assistantMsgId = json.v.message.id;
          }

          if (json.type === 'server_ste_metadata') model = json.metadata?.model_slug || null;

          const patches = Array.isArray(json.v) ? json.v : [];
          for (const p of patches) {
            if (p.o === 'append' && p.p?.includes('/message/content/parts/0')) {
              const delta = p.v;
              fullText += delta;
              if (onChunk && delta) onChunk(delta);
            }
          }
        }
      }
    });

    res.on('end', () => {
      resolve({
        reply: fullText.trim(),
        model: model || 'gpt-5-6',
        chat_id: convId,
        message_id: assistantMsgId,
        auth
      });
    });

    res.on('error', reject);
  });
}


/* =========================================================================
   4. HTTP DAEMON & REST API (PORT 18835)
   ========================================================================= */

const PORT = parseInt(process.env.HELPER_PORT || '18835', 10);
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);

  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Requested-With');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    return res.end();
  }

  if (req.method === 'GET' && url.pathname === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    return res.end(JSON.stringify({ status: 'ok', service: 'convertsw-helper-daemon', version: '1.0.0' }));
  }

  if (req.method === 'POST') {
    let bodyText = '';
    req.on('data', chunk => { bodyText += chunk; });
    req.on('end', async () => {
      let body = {};
      try { body = JSON.parse(bodyText || '{}'); } catch (_) {}

      // --- DeepSeek Routes ---
      if (url.pathname === '/deepseek/pow' || url.pathname === '/pow') {
        const token = body.token || '';
        const targetPath = body.target_path || '/api/v0/chat/completion';
        try {
          const powToken = await fetchDeepSeekPowToken(token, targetPath);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, pow_token: powToken }));
        } catch (err) {
          res.writeHead(500, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      if (url.pathname === '/deepseek/login' || url.pathname === '/login') {
        const { email, password } = body;
        if (!email || !password) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Email and password required' }));
        }
        try {
          const userInfo = await loginDeepSeek(email, password);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, ...userInfo }));
        } catch (err) {
          res.writeHead(401, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      if (url.pathname === '/deepseek/upload' || url.pathname === '/upload') {
        const { token, file_path } = body;
        if (!token || !file_path) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Token and file_path required' }));
        }
        try {
          const uploaded = await uploadDeepSeekFile(token, file_path);
          res.writeHead(200, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: true, file: uploaded }));
        } catch (err) {
          res.writeHead(500, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: err.message }));
        }
      }

      // --- Mistral Routes ---
      if (url.pathname === '/mistral/chat') {
        const prompt = body.prompt || '';
        const isStream = Boolean(body.stream);
        const webSearch = Boolean(body.web_search);

        if (!prompt) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Prompt is required' }));
        }

        if (isStream) {
          res.writeHead(200, {
            'Content-Type': 'text/event-stream',
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive'
          });

          try {
            const result = await executeMistralChat(prompt, { web_search: webSearch }, chunk => {
              res.write(`data: ${JSON.stringify({ type: 'content', text: chunk })}\n\n`);
            });
            res.write(`data: ${JSON.stringify({ type: 'done', full_text: result.reply, chat_id: result.chat_id })}\n\n`);
            res.write('data: [DONE]\n\n');
            return res.end();
          } catch (err) {
            res.write(`data: ${JSON.stringify({ type: 'error', error: err.message })}\n\n`);
            return res.end();
          }
        } else {
          try {
            const result = await executeMistralChat(prompt, { web_search: webSearch });
            res.writeHead(200, { 'Content-Type': 'application/json' });
            return res.end(JSON.stringify({ ok: true, text: result.reply, chat_id: result.chat_id, model: 'mistral-large' }));
          } catch (err) {
            res.writeHead(500, { 'Content-Type': 'application/json' });
            return res.end(JSON.stringify({ ok: false, error: err.message }));
          }
        }
      }

      // --- ChatGPT Routes ---
      if (url.pathname === '/chatgpt/chat') {
        const prompt = body.prompt || '';
        const isStream = Boolean(body.stream);
        const webSearch = Boolean(body.web_search);

        if (!prompt) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          return res.end(JSON.stringify({ ok: false, error: 'Prompt is required' }));
        }

        if (isStream) {
          res.writeHead(200, {
            'Content-Type': 'text/event-stream',
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive'
          });

          try {
            const result = await executeChatGPTChat(prompt, { webSearch }, chunk => {
              res.write(`data: ${JSON.stringify({ type: 'content', text: chunk })}\n\n`);
            });
            res.write(`data: ${JSON.stringify({ type: 'done', full_text: result.reply, chat_id: result.chat_id, model: result.model })}\n\n`);
            res.write('data: [DONE]\n\n');
            return res.end();
          } catch (err) {
            res.write(`data: ${JSON.stringify({ type: 'error', error: err.message })}\n\n`);
            return res.end();
          }
        } else {
          try {
            const result = await executeChatGPTChat(prompt, { webSearch });
            res.writeHead(200, { 'Content-Type': 'application/json' });
            return res.end(JSON.stringify({ ok: true, text: result.reply, chat_id: result.chat_id, model: result.model }));
          } catch (err) {
            res.writeHead(500, { 'Content-Type': 'application/json' });
            return res.end(JSON.stringify({ ok: false, error: err.message }));
          }
        }
      }

      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Endpoint not found' }));
    });
    return;
  }

  res.writeHead(405, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Method not allowed' }));
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`[+] ConvertSW Helper Daemon listening on http://127.0.0.1:${PORT}`);
  console.log(`[+] DeepSeek, Mistral AI, and ChatGPT reverse engines ready.`);
});
