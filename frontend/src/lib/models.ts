import { API_BASE } from "./api"
import { getAuthHeader } from "./auth"

export type ModelCapability = {
  thinking?: boolean
  search?: boolean
  vision?: boolean
  deep_research?: boolean
  image_gen?: boolean
  video_gen?: boolean
  web_dev?: boolean
  slides?: boolean
}

export type ModelOption = {
  id: string
  base_model?: string
  family?: string
  mode?: string
  display_name?: string
  capabilities?: ModelCapability
}

export type ModelGroup = {
  family: string
  models: ModelOption[]
}

export const FALLBACK_CHAT_MODELS: ModelOption[] = [
  // Combos
  { id: "combo-auto", base_model: "combo-auto", family: "Combo Models (Multi-Provider)", mode: "chat", display_name: "Combo Auto (Smart Fallback)", capabilities: { thinking: true, search: true } },
  { id: "combo-coding", base_model: "combo-coding", family: "Combo Models (Multi-Provider)", mode: "chat", display_name: "Combo Coding & Agent", capabilities: { thinking: true } },
  { id: "combo-fast", base_model: "combo-fast", family: "Combo Models (Multi-Provider)", mode: "chat", display_name: "Combo Fast (High Speed)", capabilities: {} },
  { id: "combo-reasoning", base_model: "combo-reasoning", family: "Combo Models (Multi-Provider)", mode: "thinking", display_name: "Combo Deep Reasoning", capabilities: { thinking: true } },
  // DeepSeek
  { id: "deepseek-reasoner", base_model: "deepseek-reasoner", family: "DeepSeek AI", mode: "thinking", display_name: "DeepSeek-R1 (Reasoning)", capabilities: { thinking: true } },
  { id: "deepseek-chat", base_model: "deepseek-chat", family: "DeepSeek AI", mode: "chat", display_name: "DeepSeek-V3 (Chat)", capabilities: {} },
  // ChatGPT
  { id: "gpt-5-6", base_model: "gpt-5-6", family: "ChatGPT (OpenAI)", mode: "chat", display_name: "ChatGPT 5/4o (Auto)", capabilities: { search: true } },
  { id: "gpt-4o", base_model: "gpt-4o", family: "ChatGPT (OpenAI)", mode: "chat", display_name: "GPT-4o Omnimodal", capabilities: { vision: true } },
  { id: "gpt-4o-mini", base_model: "gpt-4o-mini", family: "ChatGPT (OpenAI)", mode: "chat", display_name: "GPT-4o Mini (Fast)", capabilities: {} },
  // Mistral
  { id: "mistral-chat", base_model: "mistral-chat", family: "Mistral AI", mode: "chat", display_name: "Mistral Le Chat (Fast)", capabilities: { search: true } },
  { id: "mistral-large", base_model: "mistral-large", family: "Mistral AI", mode: "chat", display_name: "Mistral Large (Reasoning)", capabilities: { thinking: true } },
  // Qwen Official Models (Scrape Terkini)
  { id: "qwen3.8-max", base_model: "qwen3.8-max", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.8 Max (Flagship)", capabilities: { thinking: true, search: true } },
  { id: "qwen3.8-max-thinking", base_model: "qwen3.8-max", family: "Qwen AI (Alibaba)", mode: "thinking", display_name: "Qwen 3.8 Max (Thinking)", capabilities: { thinking: true } },
  { id: "qwen3.8-max-search", base_model: "qwen3.8-max", family: "Qwen AI (Alibaba)", mode: "search", display_name: "Qwen 3.8 Max (Search)", capabilities: { search: true } },
  { id: "qwen3.7-max", base_model: "qwen3.7-max", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.7 Max", capabilities: { thinking: true, search: true } },
  { id: "qwen3.7-plus", base_model: "qwen3.7-plus", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.7 Plus", capabilities: { thinking: true, search: true } },
  { id: "qwen3.6-plus", base_model: "qwen3.6-plus", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.6 Plus", capabilities: { thinking: true, search: true } },
  { id: "qwen3.5-plus", base_model: "qwen3.5-plus", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.5 Plus", capabilities: { thinking: true, search: true } },
  { id: "qwen3.5-omni-plus", base_model: "qwen3.5-omni-plus", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 3.5 Omni Plus (Multimodal)", capabilities: { vision: true } },
  { id: "qwen-max", base_model: "qwen-max", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen Max (Alias -> 3.8 Max)", capabilities: { thinking: true } },
  { id: "qwen-plus", base_model: "qwen-plus", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen Plus (Alias -> 3.7 Plus)", capabilities: { thinking: true } },
  { id: "qwen-turbo", base_model: "qwen-turbo", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen Turbo (Alias -> 3.5 Plus)", capabilities: {} },
  { id: "qwen2.5-72b-instruct", base_model: "qwen2.5-72b-instruct", family: "Qwen AI (Alibaba)", mode: "chat", display_name: "Qwen 2.5 72B Instruct", capabilities: {} },
]

export const FALLBACK_IMAGE_MODELS: ModelOption[] = [
  { id: "wanx2.1-t2i", base_model: "wanx2.1-t2i", family: "Qwen AI (Alibaba)", mode: "image", display_name: "WanX 2.1 Text-to-Image", capabilities: { image_gen: true } },
]

export const FALLBACK_VIDEO_MODELS: ModelOption[] = [
  { id: "wanx2.1-i2v", base_model: "wanx2.1-i2v", family: "Qwen AI (Alibaba)", mode: "video", display_name: "WanX 2.1 Image-to-Video", capabilities: { video_gen: true } },
]

export const CAPABILITY_LABELS: Array<{ key: keyof ModelCapability; label: string }> = [
  { key: "thinking", label: "Penalaran" },
  { key: "search", label: "Pencarian" },
  { key: "vision", label: "Visual" },
  { key: "deep_research", label: "Riset" },
  { key: "image_gen", label: "Gambar" },
  { key: "video_gen", label: "Video" },
  { key: "web_dev", label: "Web Dev" },
  { key: "slides", label: "Presentasi" },
]

const MODEL_MODE_SUFFIX_RE = /-(thinking|search|deep-research|deep_research|image|video|webdev|web-dev|slides|t2i|t2v)$/i
const TEXT_TEST_MODES = new Set(["chat", "thinking", "search", "deep_research"])
const GENERATION_MODES = new Set(["image", "video", "webdev", "slides"])
const MODE_NAME_SUFFIX: Record<string, string> = {
  thinking: "thinking",
  search: "search",
  deep_research: "deep_research",
  image: "image",
  video: "video",
  webdev: "webdev",
  slides: "slides",
}

function asText(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? value as Record<string, unknown> : {}
}

function modelMode(option: ModelOption): string {
  return option.mode || inferModeFromId(option.id)
}

function inferModeFromId(modelId: string): string {
  const id = modelId.toLowerCase()
  if (id.endsWith("-thinking")) return "thinking"
  if (id.endsWith("-search")) return "search"
  if (id.endsWith("-deep-research") || id.endsWith("-deep_research")) return "deep_research"
  if (id.endsWith("-image") || id.endsWith("-t2i")) return "image"
  if (id.endsWith("-video") || id.endsWith("-t2v")) return "video"
  if (id.endsWith("-webdev") || id.endsWith("-web-dev")) return "webdev"
  if (id.endsWith("-slides")) return "slides"
  return "chat"
}

function familyOf(option: ModelOption): string {
  if (option.family) return option.family
  const id = option.id.toLowerCase()
  if (id.startsWith("combo-")) return "Combo Models (Multi-Provider)"
  if (id.startsWith("deepseek")) return "DeepSeek AI"
  if (id.startsWith("gpt") || id.startsWith("chatgpt")) return "ChatGPT (OpenAI)"
  if (id.startsWith("mistral")) return "Mistral AI"
  if (id.startsWith("qwen") || id.startsWith("wanx")) return "Qwen AI (Alibaba)"
  const base = option.base_model || option.id.replace(MODEL_MODE_SUFFIX_RE, "")
  return base.split("-", 1)[0] || "Lainnya"
}

export function normalizeModelOption(value: unknown): ModelOption | null {
  if (typeof value === "string" && value) return { id: value, mode: inferModeFromId(value), capabilities: {} }
  const record = asRecord(value)
  const id = asText(record.id)
  if (!id) return null
  return {
    id,
    base_model: asText(record.base_model) || undefined,
    family: asText(record.family) || undefined,
    mode: asText(record.mode) || inferModeFromId(id),
    display_name: asText(record.display_name) || undefined,
    capabilities: asRecord(record.capabilities) as ModelCapability,
  }
}

export async function fetchModelOptions(): Promise<ModelOption[]> {
  const response = await fetch(`${API_BASE}/v1/models`, { headers: getAuthHeader() })
  if (!response.ok) return []
  const payload = await response.json()
  const rawItems = Array.isArray(payload?.data) ? payload.data : []
  return rawItems
    .map(normalizeModelOption)
    .filter((item: ModelOption | null): item is ModelOption => Boolean(item?.id))
}

export function isBaseModelOption(option: ModelOption): boolean {
  return option.base_model ? option.id === option.base_model : !MODEL_MODE_SUFFIX_RE.test(option.id)
}

export function isThinkingVariant(modelId: string): boolean {
  return /-thinking$/i.test(modelId) || /reasoner/i.test(modelId) || /r1/i.test(modelId)
}

export function capabilityBadges(option?: ModelOption): string[] {
  if (!option?.capabilities) return []
  return CAPABILITY_LABELS.filter(item => option.capabilities?.[item.key]).map(item => item.label)
}

export function filterTextTestModels(options: ModelOption[]): ModelOption[] {
  const filtered = options.filter(option => {
    const mode = modelMode(option)
    return TEXT_TEST_MODES.has(mode) && !GENERATION_MODES.has(mode)
  })
  const baseModels = filtered.filter(option => modelMode(option) === "chat")
  const existingIds = new Set(filtered.map(option => option.id))
  const searchVariants = baseModels
    .map(option => ({
      ...option,
      id: option.id.endsWith("-search") ? option.id : `${option.id}-search`,
      base_model: option.base_model || option.id,
      mode: "search",
      display_name: `${option.display_name || option.id} search`,
      capabilities: { search: true },
    }))
    .filter(option => !existingIds.has(option.id))
  const withSearch = [...filtered, ...searchVariants]
  return withSearch.length ? withSearch : FALLBACK_CHAT_MODELS
}

export function filterImageModels(options: ModelOption[]): ModelOption[] {
  const explicit = options.filter(option => modelMode(option) === "image")
  if (explicit.length) return explicit
  const capable = options
    .filter(option => option.capabilities?.image_gen && !GENERATION_MODES.has(modelMode(option)))
    .map(option => ({
      ...option,
      id: option.id.endsWith("-image") ? option.id : `${option.id}-image`,
      base_model: option.base_model || option.id,
      mode: "image",
      display_name: `${option.display_name || option.id} image`,
      capabilities: { image_gen: true },
    }))
  return capable.length ? capable : FALLBACK_IMAGE_MODELS
}

export function filterVideoModels(options: ModelOption[]): ModelOption[] {
  const explicit = options.filter(option => modelMode(option) === "video")
  if (explicit.length) return explicit
  const capable = options
    .filter(option => option.capabilities?.video_gen && !GENERATION_MODES.has(modelMode(option)))
    .map(option => ({
      ...option,
      id: option.id.endsWith("-video") ? option.id : `${option.id}-video`,
      base_model: option.base_model || option.id,
      mode: "video",
      display_name: `${option.display_name || option.id} video`,
      capabilities: { video_gen: true },
    }))
  return capable.length ? capable : FALLBACK_VIDEO_MODELS
}

export function chooseDefaultModel(options: ModelOption[], currentModel?: string, preferredId?: string): string {
  if (preferredId && options.some(option => option.id === preferredId)) return preferredId
  if (currentModel && options.some(option => option.id === currentModel)) return currentModel
  const comboAuto = options.find(o => o.id === "combo-auto")
  if (comboAuto) return comboAuto.id
  const base = options.find(isBaseModelOption)
  return base?.id || options[0]?.id || "combo-auto"
}

export function groupModelOptions(options: ModelOption[]): ModelGroup[] {
  const groups = new Map<string, ModelOption[]>()
  options.forEach(option => {
    const family = familyOf(option)
    groups.set(family, [...(groups.get(family) || []), option])
  })
  return Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([family, models]) => ({
      family,
      models: models.sort((a, b) => a.id.localeCompare(b.id)),
    }))
}

export function formatModeLabel(mode?: string): string {
  switch (mode) {
    case "thinking": return "Penalaran"
    case "search": return "Pencarian"
    case "deep_research": return "Riset"
    case "image": return "Gambar"
    case "video": return "Video"
    case "webdev": return "Web Dev"
    case "slides": return "Presentasi"
    default: return "Chat"
  }
}

export function formatModelName(option: ModelOption): string {
  const mode = modelMode(option)
  const suffix = MODE_NAME_SUFFIX[mode]
  const rawName = option.display_name || option.id
  const name = suffix ? rawName.replace(new RegExp(`\\s+${suffix}$`, "i"), "") : rawName
  return name === option.id ? option.id : `${name} (${option.id})`
}

export function formatModelOptionLabel(option: ModelOption): string {
  return `${formatModelName(option)} · ${formatModeLabel(modelMode(option))}`
}
