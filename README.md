# 🚀 ConvertSW-Gateway (Router AI Wrap)

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8.svg?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.24+" />
  <img src="https://img.shields.io/badge/React-19-61DAFB.svg?style=for-the-badge&logo=react&logoColor=black" alt="React 19" />
  <img src="https://img.shields.io/badge/Node.js-20+-339933.svg?style=for-the-badge&logo=nodedotjs&logoColor=white" alt="Node.js 20+" />
  <img src="https://img.shields.io/badge/API-OpenAI%20%7C%20Anthropic%20%7C%20Gemini-green.svg?style=for-the-badge" alt="Multi Protocol" />
  <img src="https://img.shields.io/badge/Router-Smart%20Fallback%20%26%20Round%20Robin-purple.svg?style=for-the-badge" alt="Router Engine" />
  <img src="https://img.shields.io/badge/Platform-Android%20Termux%20%7C%20Linux%20VPS-blue.svg?style=for-the-badge" alt="Platform" />
</p>

**ConvertSW-Gateway** adalah router gateway AI multi-provider berkinerja tinggi (*Unified AI Router & Wrapper*) yang menggabungkan berbagai reverse provider AI unggulan ke dalam satu endpoint standar yang kompatibel penuh dengan **OpenAI**, **Anthropic**, dan **Gemini API**.

Gateway ini menyatukan 4 ekosistem AI besar:
1. **DeepSeek AI** (DeepSeek-V3 & DeepSeek-R1 Deep Reasoning via Wasm SHA-3 PoW)
2. **ChatGPT / OpenAI** (GPT-5/4o via Sentinel FNV-1a PoW & Turnstile bypass)
3. **Mistral AI** (Mistral Large Reasoning, Mistral Chat & Codestral via Le Chat mobile reverse)
4. **Qwen AI / Alibaba Cloud** (Qwen 2.5 Max, Qwen 2.5 72B Instruct, WanX Multimodal via Anti-WAF bypass)

---

## 🌟 Fitur Utama

- 🔄 **Combo Model & Auto-Rolling**: Gabungkan beberapa model dan provider ke dalam satu nama virtual model (misal `combo-auto`, `combo-coding`).
  - **Smart Fallback (Failover Otomatis)**: Jika Provider A mengalami limit (429), timeout, atau internal server error (500), router secara transparan mengalihkan eksekusi ke Provider B tanpa memutus koneksi streaming pengguna.
  - **Round-Robin Rolling**: Memutar target penyedia secara bergiliran pada setiap panggilan baru untuk pemerataan kuota dan beban trafik.
- 🛠️ **Universal Tool / Function Calling (100% Works)**: Mesin `ToolSieve` di sisi backend Go memproses output streaming dari seluruh provider dan menormalisasikannya ke format resmi OpenAI `finish_reason: "tool_calls"`. Mendukung autonomous coding agent seperti **Cline**, **Roo Code**, **Cursor**, **Dify**, dan **LangChain**.
- 📎 **File Mention & Local Attachment**: Dukungan parsing sintaks `@/path/to/file` dan blok attachment lokal yang disuntikkan secara aman ke dalam konteks percakapan tanpa bergantung pada API upload pihak ketiga.
- 🌐 **Web Search Terintegrasi**: Mendukung pencarian web langsung untuk model ChatGPT (`chatgpt-search`) dan Mistral (`mistral-search`).
- ⚡ **Super Ringan & Hemat RAM (< 50MB)**: Inti gateway ditulis dalam bahasa Go murni berkecepatan tinggi, didampingi 1 daemon Node.js terpadu untuk PoW bypass. Sangat ideal dijalankan di Android Termux maupun server VPS berspesifikasi rendah.
- 🎨 **WebUI Management Console (100% Bahasa Indonesia)**: Dashboard interaktif berbasis React 19 + Tailwind CSS untuk memantau status sistem, membuat model combo kustom, menguji percakapan secara langsung, mengelola token API, dan melihat log aktivitas real-time.

---

## 📋 Model Bawaan & Model Combo

### 1. Model Combo Virtual (Multi-Provider Router)
| Model ID | Strategi | Rantai Target Eksekusi |
|---|---|---|
| `combo-auto` | Smart Fallback | `deepseek-reasoner` &rarr; `gpt-5-6` &rarr; `qwen-max` &rarr; `mistral-large` |
| `combo-coding` | Smart Fallback | `deepseek-reasoner` &rarr; `gpt-5-6` &rarr; `qwen2.5-72b-instruct` &rarr; `mistral-large` |
| `combo-fast` | Smart Fallback | `mistral-chat` &rarr; `qwen-turbo` &rarr; `gpt-4o-mini` &rarr; `deepseek-chat` |
| `combo-reasoning` | Smart Fallback | `deepseek-reasoner` &rarr; `gpt-5-6` &rarr; `qwen-max` |

*Anda dapat membuat model combo baru dan mengatur strategi (Fallback / Round-Robin) sesuka hati melalui menu **Model Combo** di WebUI.*

### 2. Model Mandiri Per Provider
- **DeepSeek AI**: `deepseek-chat`, `deepseek-reasoner`, `deepseek-v3`, `deepseek-r1`, `deepseek-vision`
- **ChatGPT (OpenAI)**: `gpt-5-6`, `gpt-4o`, `gpt-4o-mini`, `chatgpt-auto`, `chatgpt-search`
- **Mistral AI**: `mistral-chat`, `mistral-large`, `mistral-small`, `mistral-vibe`, `mistral-search`
- **Qwen AI**: `qwen-max`, `qwen-plus`, `qwen-turbo`, `qwen2.5-72b-instruct`, `wanx2.1-t2i`, `wanx2.1-i2v`

---

## ⚡ Instalasi & Menjalankan Gateway

### 1. Prasyarat Sistem
- **Golang 1.22+**
- **Node.js 20+**
- **Linux** (Ubuntu, Debian, Termux Android)

### 2. Jalankan Layanan

```bash
cd /root/termux_home/ConvertSW-Gateway

# Jalankan dalam mode daemon background
./start.sh -d

# Periksa status log live
tail -f logs/output.log

# Menghentikan layanan
./stop.sh
```

- **Web Dashboard**: Buka browser di **`http://localhost:7862`**
- **Admin Key Default**: `admin123456`
- **Port Gateway Go**: `7862`
- **Port Node Helper Daemon**: `18835`

---

## 🔌 Kompatibilitas Endpoint API

| Protokol | Endpoint | Keterangan |
|---|---|---|
| **OpenAI Chat** | `POST /v1/chat/completions` | Streaming SSE, Tool Calling, Reasoning Content |
| **OpenAI Models** | `GET /v1/models` | Daftar semua model combo dan provider |
| **Anthropic Claude** | `POST /v1/messages` | Kompatibel Claude CLI & Cursor Anthropic Mode |
| **Google Gemini** | `POST /v1beta/models/{model}:generateContent` | Kompatibel Gemini SDK |
| **Image Generation** | `POST /v1/images/generations` | WanX 2.1 Text-to-Image |
| **Video Generation** | `POST /v1/videos/generations` | WanX 2.1 Image-to-Video |

---

## 🤖 Contoh Pemanggilan API

### Menggunakan cURL (Model Combo dengan Auto-Fallback)

```bash
curl -X POST http://localhost:7862/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer admin123456" \
  -d '{
    "model": "combo-auto",
    "messages": [
      {"role": "user", "content": "Halo! Siapa namamu dan provider apa yang sedang aktif?"}
    ],
    "stream": true
  }'
```

### Menggunakan Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    api_key="admin123456",
    base_url="http://localhost:7862/v1"
)

response = client.chat.completions.create(
    model="combo-coding",
    messages=[
        {"role": "system", "content": "Anda adalah asisten coding ahli."},
        {"role": "user", "content": "Buatkan fungsi Go untuk membalik string Unicode."}
    ],
    stream=True
)

for chunk in response:
    content = chunk.choices[0].delta.content or ""
    print(content, end="", flush=True)
```

---

## 🛡️ Lisensi
GPL-3.0 License.
